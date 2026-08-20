package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrAdjustCallbackDisabled     = errors.New("Adjust attribution is disabled")
	ErrAdjustCallbackUnauthorized = errors.New("invalid Adjust callback token")
	ErrAdjustCallbackInvalid      = errors.New("invalid Adjust attribution callback")
	ErrAdjustAppReportInvalid     = errors.New("invalid APP Adjust attribution report")
	ErrAdjustAttributionConflict  = errors.New("Adjust attribution conflicts with an existing user or ADID binding")
)

type AdjustAppReportRequest struct {
	TrackerToken      string           `json:"trackerToken" binding:"omitempty,max=64"`
	TrackerName       string           `json:"trackerName" binding:"omitempty"`
	Campaign          string           `json:"campaign" binding:"omitempty,max=255"`
	Network           string           `json:"network" binding:"omitempty,max=255"`
	Creative          string           `json:"creative" binding:"omitempty,max=255"`
	Adgroup           string           `json:"adgroup" binding:"omitempty,max=255"`
	ClickLabel        string           `json:"clickLabel" binding:"omitempty,max=512"`
	CostType          string           `json:"costType" binding:"omitempty,max=32"`
	CostAmount        AdjustCostAmount `json:"costAmount" binding:"omitempty,max=64"`
	CostCurrency      string           `json:"costCurrency" binding:"omitempty,max=16"`
	FBInstallReferrer string           `json:"fbInstallReferrer" binding:"omitempty"`
	GoogleAdID        string           `json:"googleAdId" binding:"omitempty,max=128"`
	ADID              string           `json:"adid" binding:"required,max=64"`
	IDFA              string           `json:"idfa" binding:"omitempty,max=128"`
	IDFV              string           `json:"idfv" binding:"omitempty,max=128"`
}

// AdjustCostAmount accepts the current SDK number/null representation while
// remaining backward compatible with older clients that sent "NaN".
type AdjustCostAmount string

func (value *AdjustCostAmount) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*value = ""
		return nil
	}
	if strings.HasPrefix(raw, "\"") {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		*value = AdjustCostAmount(text)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return errors.New("costAmount must be a number, string, or null")
	}
	*value = AdjustCostAmount(number.String())
	return nil
}

type AdjustAppReportResult struct {
	ADID    string `json:"adid"`
	Fused   bool   `json:"fused"`
	Applied bool   `json:"applied"`
	Status  uint32 `json:"status"`
}

type AdjustCallbackInput struct {
	Token      string
	Payload    map[string]any
	ReceivedAt time.Time
}

type AdjustCallbackResult struct {
	Duplicate bool   `json:"duplicate"`
	Matched   bool   `json:"matched"`
	Applied   bool   `json:"applied"`
	Status    uint32 `json:"status"`
}

type AdjustAttributionService struct {
	repo            *repository.AdjustAttributionRepo
	channelRepo     *repository.ChannelRepo
	userRepo        *repository.AppUserRepo
	attributionRepo *repository.UserAttributionRepo
	mediaRepo       *repository.MediaRepo
}

type adjustCallbackClass uint8

const (
	adjustCallbackIgnored adjustCallbackClass = iota
	adjustCallbackInitialInstall
	adjustCallbackInstallUpdate
	adjustCallbackReattribution
	adjustCallbackGDPRForget
)

func NewAdjustAttributionService() *AdjustAttributionService {
	return &AdjustAttributionService{
		repo:            repository.NewAdjustAttributionRepo(),
		channelRepo:     repository.NewChannelRepo(),
		userRepo:        repository.NewAppUserRepo(),
		attributionRepo: repository.NewUserAttributionRepo(),
		mediaRepo:       repository.NewMediaRepo(),
	}
}

func (s *AdjustAttributionService) resolveAttributionDimensions(ctx context.Context, item *model.VideoAdjustAttribution) error {
	mediaID, _, err := s.repo.ResolveMedia(ctx, item.Network)
	if err != nil {
		return err
	}
	item.MediaID = mediaID

	channelID := uint64(0)
	if item.IsOrganic == 0 && item.AdAccountID != 0 {
		matchedID, err := s.channelRepo.GetByAdAccountID(ctx, item.AdAccountID, mediaID)
		switch {
		case err == nil:
			channelID = matchedID
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}
	}
	if channelID == 0 {
		defaultID, err := s.channelRepo.GetDefaultID(ctx)
		if err != nil {
			return err
		}
		channelID = defaultID
	}
	item.ChannelID = channelID
	return nil
}

func (s *AdjustAttributionService) resolvedAttributionChannelCode(ctx context.Context, item *model.VideoAdjustAttribution) (string, error) {
	if item.IsOrganic != 0 {
		return "", nil
	}
	if item.ChannelID != 0 {
		channel, err := s.channelRepo.GetEnabledByID(ctx, item.ChannelID)
		if err == nil {
			return channel.ChannelCode, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}
	return attributionChannelCode(item), nil
}

// ReportApp idempotently binds an authenticated user to an ADID. The Adjust
// SDK attribution-changed callback may report the same ADID repeatedly, and a
// user may own multiple ADIDs. An ADID can never be rebound to another user.
func (s *AdjustAttributionService) ReportApp(ctx context.Context, userID uint64, req AdjustAppReportRequest) (*AdjustAppReportResult, error) {
	if !config.Cfg.Adjust.Enabled {
		return nil, ErrAdjustCallbackDisabled
	}
	report, rawPayload, err := normalizeAdjustAppReport(req)
	if err != nil {
		return nil, err
	}
	result := &AdjustAppReportResult{ADID: report.ADID}
	err = repository.Transaction(ctx, func(txCtx context.Context) error {
		fusion, err := s.repo.GetByADID(txCtx, report.ADID, true)
		now := time.Now()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user, lockErr := s.userRepo.GetByIDForUpdate(txCtx, userID)
			if lockErr != nil {
				return lockErr
			}
			fusion = appReportFusion(report, now)
			if err = s.resolveAttributionDimensions(txCtx, fusion); err != nil {
				return err
			}
			payload := string(rawPayload)
			fusion.UserID = userID
			fusion.AppCode = strings.TrimSpace(user.AppName)
			fusion.AppPayload = &payload
			fusion.AppReportedAt = &now
			fusion.MatchStatus = repository.AdjustMatchStatusPendingApp
			fusion.CreatedAt, fusion.UpdatedAt = now, now
			if err = s.repo.Create(txCtx, fusion); err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return ErrAdjustAttributionConflict
				}
				return err
			}
			result.Status = fusion.MatchStatus
			return nil
		}
		if err != nil {
			return err
		}
		if fusion.UserID != 0 && fusion.UserID != userID {
			return ErrAdjustAttributionConflict
		}
		wasUnbound := fusion.UserID == 0
		user, err := s.userRepo.GetByIDForUpdate(txCtx, userID)
		if err != nil {
			return err
		}

		applyAppReport(fusion, report)
		if fusion.MatchStatus == repository.AdjustMatchStatusPendingApp {
			if err = s.resolveAttributionDimensions(txCtx, fusion); err != nil {
				return err
			}
		}
		payload := string(rawPayload)
		fusion.UserID = userID
		fusion.AppCode = strings.TrimSpace(user.AppName)
		fusion.AppPayload = &payload
		fusion.AppReportedAt = &now
		fusion.UpdatedAt = now
		if fusion.MatchStatus == repository.AdjustMatchStatusPendingApp {
			if err = s.repo.Update(txCtx, fusion.ID, fusionUpdates(fusion)); err != nil {
				return err
			}
			result.Status = fusion.MatchStatus
			return nil
		}
		callbackClass := classifyAdjustCallback(fusion)
		applied, err := s.completeFusion(txCtx, fusion, user, fusion, callbackClass, wasUnbound)
		if err != nil {
			return err
		}
		result.Status = fusion.MatchStatus
		result.Fused = true
		result.Applied = applied
		return nil
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		err = ErrAdjustAttributionConflict
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Handle merges every server callback into the ADID row in
// video_adjust_attribution. A missing APP report leaves that row pending; the
// later authenticated report completes it.
func (s *AdjustAttributionService) Handle(ctx context.Context, input AdjustCallbackInput) (*AdjustCallbackResult, error) {
	callback, err := normalizeAdjustCallback(input)
	if err != nil {
		return nil, err
	}
	result := &AdjustCallbackResult{}
	err = repository.Transaction(ctx, func(txCtx context.Context) error {
		callbackClass := classifyAdjustCallback(callback)
		if callbackClass == adjustCallbackGDPRForget {
			if err = s.repo.ForgetDevice(txCtx, callback.AdjustADID); err != nil {
				return err
			}
			result.Status = 0
			return nil
		}

		fusion, err := s.repo.GetByADID(txCtx, callback.AdjustADID, true)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fusion = callbackFusion(callback)
			if callbackClass == adjustCallbackIgnored {
				fusion.MatchStatus = fusion.MatchStatus
			} else {
				if err = s.resolveAttributionDimensions(txCtx, fusion); err != nil {
					return err
				}
				fusion.MatchStatus = repository.AdjustMatchStatusPendingCallback
			}
			if err = s.repo.Create(txCtx, fusion); err != nil {
				return err
			}
			result.Status = fusion.MatchStatus
			return nil
		}
		if err != nil {
			return err
		}

		if callbackClass == adjustCallbackIgnored {
			recordCallbackReceipt(fusion, callback)
			if err = s.repo.Update(txCtx, fusion.ID, map[string]any{
				"callback_count": fusion.CallbackCount, "last_callback_key": fusion.LastCallbackKey,
				"callback_payload":     fusion.CallbackPayload,
				"callback_received_at": fusion.CallbackReceivedAt, "updated_at": fusion.UpdatedAt,
			}); err != nil {
				return err
			}
			result.Matched = fusion.UserID != 0
			result.Status = fusion.MatchStatus
			return nil
		}

		incomingAt := adjustCallbackTime(callback)
		stale := fusion.CallbackReceivedAt != nil && fusion.AdjustCreatedAt != nil &&
			fusion.AdjustCreatedAt.After(incomingAt)
		acquisition := fusion
		if stale {
			recordCallbackReceipt(fusion, callback)
			acquisition = callback
			if err = s.resolveAttributionDimensions(txCtx, acquisition); err != nil {
				return err
			}
		} else {
			mergeCallbackIntoFusion(fusion, callback)
			if err = s.resolveAttributionDimensions(txCtx, fusion); err != nil {
				return err
			}
		}

		if fusion.UserID == 0 {
			fusion.MatchStatus = repository.AdjustMatchStatusPendingApp
			if err := s.repo.Update(txCtx, fusion.ID, fusionUpdates(fusion)); err != nil {
				return err
			}
			result.Status = fusion.MatchStatus
			return nil
		}

		user, err := s.userRepo.GetByIDForUpdate(txCtx, fusion.UserID)
		if err != nil {
			return err
		}
		applied, err := s.completeFusion(txCtx, fusion, user, acquisition, callbackClass, true)
		if err != nil {
			return err
		}
		result.Matched, result.Applied = true, applied
		result.Status = fusion.MatchStatus
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *AdjustAttributionService) completeFusion(ctx context.Context, fusion *model.VideoAdjustAttribution, user *model.VideoUser, acquisition *model.VideoAdjustAttribution, callbackClass adjustCallbackClass, applyAcquisition bool) (bool, error) {
	now := time.Now()
	fusion.MatchStatus = repository.AdjustMatchStatusFused
	if fusion.FusedAt == nil {
		fusion.FusedAt = &now
	}
	if strings.TrimSpace(fusion.AppCode) == "" {
		fusion.AppCode = strings.TrimSpace(user.AppName)
	}
	fusion.UpdatedAt = now
	if err := s.repo.Update(ctx, fusion.ID, fusionUpdates(fusion)); err != nil {
		return false, err
	}

	applied := false
	var err error
	if applyAcquisition && (callbackClass == adjustCallbackInitialInstall || callbackClass == adjustCallbackInstallUpdate) {
		applied, err = s.applyInitialAcquisition(
			ctx, acquisition, user,
			callbackClass == adjustCallbackInstallUpdate,
		)
		if err != nil {
			return false, err
		}
	}
	if applied && fusion.SummaryApplied == 0 {
		fusion.SummaryApplied = 1
		fusion.UpdatedAt = time.Now()
		if err = s.repo.Update(ctx, fusion.ID, map[string]any{
			"summary_applied": fusion.SummaryApplied, "updated_at": fusion.UpdatedAt,
		}); err != nil {
			return false, err
		}
	}
	return applied, nil
}

func (s *AdjustAttributionService) applyInitialAcquisition(ctx context.Context, callback *model.VideoAdjustAttribution, user *model.VideoUser, allowCorrection bool) (bool, error) {
	channelCode, err := s.resolvedAttributionChannelCode(ctx, callback)
	if err != nil {
		return false, err
	}

	attributedAt := initialAttributionTime(callback)
	appCode := strings.TrimSpace(callback.AppCode)
	if appCode == "" {
		appCode = strings.TrimSpace(user.AppName)
	}
	applied, err := s.attributionRepo.ApplyFusedAttribution(ctx, &repository.FusedUserAttribution{
		AppCode: appCode, UserID: user.ID, AdjustADID: callback.AdjustADID,
		ChannelID: callback.ChannelID, MediaID: callback.MediaID, IMEI: user.DeviceCode,
		GoogleAdID: callback.GoogleAdID, ActivityKind: callback.ActivityKind,
		AttributionType: callback.AttributionType, IsOrganic: callback.IsOrganic,
		Reattributed: callback.Reattributed, IsRedownload: callback.IsRedownload,
		ClickTime: callback.ClickTime, InstallTime: callback.InstallTime,
		AttributedAt: &attributedAt, ReattributedAt: callback.ReattributedAt,
		AttributionUpdatedAt: callback.AttributionUpdatedAt, AdjustCreatedAt: callback.AdjustCreatedAt,
	}, allowCorrection)
	if err != nil {
		return false, err
	}
	if !applied {
		return false, nil
	}

	userUpdates := map[string]any{"attribution_clicked_at": nil}
	if callback.IsOrganic != 0 {
		userUpdates["channel_id"] = ""
	} else {
		userUpdates["channel_id"] = channelCode
		if callback.ClickTime != nil {
			userUpdates["attribution_clicked_at"] = *callback.ClickTime
		}
	}
	if err := s.userRepo.Update(ctx, user.ID, userUpdates); err != nil {
		return false, err
	}
	return true, nil
}

func normalizeAdjustAppReport(req AdjustAppReportRequest) (AdjustAppReportRequest, []byte, error) {
	req.TrackerToken = strings.TrimSpace(req.TrackerToken)
	req.TrackerName = strings.TrimSpace(req.TrackerName)
	req.Campaign = strings.TrimSpace(req.Campaign)
	req.Network = strings.TrimSpace(req.Network)
	req.Creative = strings.TrimSpace(req.Creative)
	req.Adgroup = strings.TrimSpace(req.Adgroup)
	req.ClickLabel = strings.TrimSpace(req.ClickLabel)
	req.CostType = strings.TrimSpace(req.CostType)
	req.CostAmount = AdjustCostAmount(strings.TrimSpace(string(req.CostAmount)))
	req.CostCurrency = strings.TrimSpace(req.CostCurrency)
	req.FBInstallReferrer = strings.TrimSpace(req.FBInstallReferrer)
	req.GoogleAdID = strings.TrimSpace(req.GoogleAdID)
	req.ADID = strings.ToLower(strings.TrimSpace(req.ADID))
	req.IDFA = strings.TrimSpace(req.IDFA)
	req.IDFV = strings.TrimSpace(req.IDFV)
	if req.ADID == "" {
		return req, nil, fmt.Errorf("%w: adid is required", ErrAdjustAppReportInvalid)
	}
	fields := map[string]struct {
		value string
		max   int
	}{
		"trackerToken": {req.TrackerToken, 64},
		"campaign":     {req.Campaign, 255}, "network": {req.Network, 255},
		"creative": {req.Creative, 255}, "adgroup": {req.Adgroup, 255},
		"clickLabel": {req.ClickLabel, 512}, "costType": {req.CostType, 32},
		"costAmount": {string(req.CostAmount), 64}, "costCurrency": {req.CostCurrency, 16},
		"googleAdId": {req.GoogleAdID, 128},
		"adid":       {req.ADID, 64}, "idfa": {req.IDFA, 128}, "idfv": {req.IDFV, 128},
	}
	for name, field := range fields {
		if len(field.value) > field.max {
			return req, nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrAdjustAppReportInvalid, name, field.max)
		}
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return req, nil, fmt.Errorf("%w: payload is not valid JSON", ErrAdjustAppReportInvalid)
	}
	if limit := config.Cfg.Adjust.MaxBodyBytes; limit > 0 && int64(len(payload)) > limit {
		return req, nil, fmt.Errorf("%w: payload exceeds configured size", ErrAdjustAppReportInvalid)
	}
	return req, payload, nil
}

func normalizeAdjustCallback(input AdjustCallbackInput) (*model.VideoAdjustAttribution, error) {
	cfg := config.Cfg.Adjust
	if !cfg.Enabled {
		return nil, ErrAdjustCallbackDisabled
	}
	//if !secureTokenEqual(input.Token, cfg.CallbackToken) {
	//	return nil, ErrAdjustCallbackUnauthorized
	//}
	payload := sanitizedAdjustPayload(input.Payload)
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: payload is not valid JSON", ErrAdjustCallbackInvalid)
	}
	if cfg.MaxBodyBytes > 0 && int64(len(rawPayload)) > cfg.MaxBodyBytes {
		return nil, fmt.Errorf("%w: payload exceeds configured size", ErrAdjustCallbackInvalid)
	}

	adid := strings.ToLower(adjustParam(payload, "adid", "adjust_adid"))
	if adid == "" {
		return nil, fmt.Errorf("%w: adid is required", ErrAdjustCallbackInvalid)
	}
	clickTime, err := adjustTimeParam(payload, "click_time", "click_time_unix", "click_time_unix_ms")
	if err != nil {
		return nil, err
	}
	installTime, err := adjustTimeParam(payload, "install_time", "installed_at", "install_time_unix", "install_time_unix_ms")
	if err != nil {
		return nil, err
	}
	createdAt, err := adjustTimeParam(payload, "created_at", "created_at_unix", "created_at_unix_ms")
	if err != nil {
		return nil, err
	}
	reattributedAt, err := adjustTimeParam(payload, "reattributed_at", "reattributed_at_unix", "reattributed_at_unix_ms")
	if err != nil {
		return nil, err
	}
	attributionUpdatedAt, err := adjustTimeParam(
		payload, "attribution_updated_at", "attribution_updated_at_unix", "attribution_updated_at_unix_ms",
	)
	if err != nil {
		return nil, err
	}

	trackerToken := adjustParam(payload, "tracker_token", "tracker")
	trackerName := adjustParam(payload, "tracker_name")
	appToken := adjustParam(payload, "app_token")
	outdatedTracker := adjustParam(payload, "outdated_tracker", "outdated_tracker_token")
	outdatedTrackerName := adjustParam(payload, "outdated_tracker_name")
	network := adjustParam(payload, "network", "network_name")
	activityKind := adjustParam(payload, "activity_kind")
	gpsAdid := adjustParam(payload, "gps_adid")
	idfa := adjustParam(payload, "idfa")
	idfv := adjustParam(payload, "idfv")
	adgroup := adjustParam(payload, "adgroup_name")
	campaign := adjustParam(payload, "campaign_name")
	creative := adjustParam(payload, "creative_name")
	ipAddress := adjustParam(payload, "ip_address")
	userAgent := adjustParam(payload, "user_agent")
	if activityKind == "" {
		activityKind = "attribution"
	}
	organic := adjustBoolParam(payload, "is_organic") || strings.EqualFold(network, "organic") || strings.EqualFold(trackerName, "organic")
	reattributed := adjustBoolParam(payload, "reattributed", "is_reattributed") || strings.Contains(strings.ToLower(activityKind), "reattribut")
	attributionType := adjustParam(payload, "attribution_type")
	if attributionType == "" {
		switch {
		case organic:
			attributionType = "organic"
		case reattributed:
			attributionType = "reattribution"
		default:
			attributionType = "attribution"
		}
	}

	lengths := map[string]struct {
		value string
		max   int
	}{
		"adid":                  {adid, 64},
		"tracker_token":         {trackerToken, 64},
		"gps_adid":              {gpsAdid, 128},
		"idfa":                  {idfa, 128},
		"idfv":                  {idfv, 128},
		"tracker_name":          {trackerName, 255},
		"network":               {network, 255},
		"app_token":             {appToken, 64},
		"outdated_tracker":      {outdatedTracker, 64},
		"outdated_tracker_name": {outdatedTrackerName, 255},
		"campaign":              {campaign, 255},
		"adgroup":               {adgroup, 255},
		"creative":              {creative, 255},
		"activity_kind":         {activityKind, 64},
		"attribution_type":      {attributionType, 32},
		"ip_address":            {ipAddress, 64},
		"user_agent":            {userAgent, 1024},
	}
	for name, field := range lengths {
		if len(field.value) > field.max {
			return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrAdjustCallbackInvalid, name, field.max)
		}
	}

	receivedAt := input.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now()
	}
	keyHash := sha256.Sum256(rawPayload)
	payloadText := string(rawPayload)
	return &model.VideoAdjustAttribution{
		LastCallbackKey:      hex.EncodeToString(keyHash[:]),
		AdjustADID:           adid,
		TrackerToken:         trackerToken,
		TrackerName:          trackerName,
		AppToken:             appToken,
		OutdatedTracker:      outdatedTracker,
		OutdatedTrackerName:  outdatedTrackerName,
		Network:              network,
		Campaign:             campaign,
		Adgroup:              adgroup,
		Creative:             creative,
		ActivityKind:         activityKind,
		AttributionType:      attributionType,
		IsOrganic:            boolUint8(organic),
		Reattributed:         boolUint8(reattributed),
		IsRedownload:         boolUint8(adjustBoolParam(payload, "is_redownload", "redownload")),
		DeviceIP:             ipAddress,
		UserAgent:            userAgent,
		ClickTime:            clickTime,
		InstallTime:          installTime,
		ReattributedAt:       reattributedAt,
		AttributionUpdatedAt: attributionUpdatedAt,
		AdAccountID:          getNetworkAccountID(payload, network),
		GoogleAdID:           gpsAdid,
		IDFA:                 idfa,
		IDFV:                 idfv,
		AdjustCreatedAt:      createdAt,
		MatchStatus:          repository.AdjustMatchStatusPendingApp,
		CallbackPayload:      &payloadText,
		CallbackReceivedAt:   &receivedAt,
		CreatedAt:            receivedAt,
		UpdatedAt:            receivedAt,
	}, nil
}

func appReportFusion(req AdjustAppReportRequest, now time.Time) *model.VideoAdjustAttribution {
	organic := strings.EqualFold(req.Network, "organic") || strings.EqualFold(req.TrackerName, "organic")
	return &model.VideoAdjustAttribution{
		AdjustADID:   req.ADID,
		TrackerToken: req.TrackerToken, TrackerName: req.TrackerName,
		Network: req.Network, Campaign: req.Campaign, Adgroup: req.Adgroup, Creative: req.Creative,
		ClickLabel: req.ClickLabel, CostType: req.CostType, CostAmount: string(req.CostAmount),
		CostCurrency: req.CostCurrency, FBInstallReferrer: req.FBInstallReferrer,
		GoogleAdID: req.GoogleAdID, IDFA: req.IDFA, IDFV: req.IDFV,
		IsOrganic: boolUint8(organic), CreatedAt: now, UpdatedAt: now,
	}
}

func applyAppReport(fusion *model.VideoAdjustAttribution, req AdjustAppReportRequest) {
	callbackAuthoritative := hasActionableAdjustCallback(fusion)
	fusion.ClickLabel, fusion.CostType = req.ClickLabel, req.CostType
	fusion.CostAmount, fusion.CostCurrency = string(req.CostAmount), req.CostCurrency
	assignIfNotEmpty(&fusion.FBInstallReferrer, req.FBInstallReferrer)
	assignIfNotEmpty(&fusion.GoogleAdID, req.GoogleAdID)
	assignIfNotEmpty(&fusion.IDFA, req.IDFA)
	assignIfNotEmpty(&fusion.IDFV, req.IDFV)
	if callbackAuthoritative {
		return
	}
	fusion.TrackerToken, fusion.TrackerName = req.TrackerToken, req.TrackerName
	fusion.Network, fusion.Campaign = req.Network, req.Campaign
	fusion.Adgroup, fusion.Creative = req.Adgroup, req.Creative
	fusion.IsOrganic = boolUint8(strings.EqualFold(req.Network, "organic") || strings.EqualFold(req.TrackerName, "organic"))
}

func hasActionableAdjustCallback(item *model.VideoAdjustAttribution) bool {
	if item == nil || strings.TrimSpace(item.LastCallbackKey) == "" {
		return false
	}
	switch classifyAdjustCallback(item) {
	case adjustCallbackInitialInstall, adjustCallbackInstallUpdate, adjustCallbackReattribution:
		return true
	default:
		return false
	}
}

func callbackFusion(callback *model.VideoAdjustAttribution) *model.VideoAdjustAttribution {
	now := time.Now()
	fusion := &model.VideoAdjustAttribution{
		AdjustADID: callback.AdjustADID, MatchStatus: repository.AdjustMatchStatusPendingApp,
		CreatedAt: now, UpdatedAt: now,
	}
	mergeCallbackIntoFusion(fusion, callback)
	fusion.CreatedAt = now
	return fusion
}

func mergeCallbackIntoFusion(fusion *model.VideoAdjustAttribution, callback *model.VideoAdjustAttribution) {
	assignIfNotEmpty(&fusion.TrackerToken, callback.TrackerToken)
	assignIfNotEmpty(&fusion.TrackerName, callback.TrackerName)
	assignIfNotEmpty(&fusion.AppToken, callback.AppToken)
	assignIfNotEmpty(&fusion.OutdatedTracker, callback.OutdatedTracker)
	assignIfNotEmpty(&fusion.OutdatedTrackerName, callback.OutdatedTrackerName)
	assignIfNotEmpty(&fusion.Network, callback.Network)
	assignIfNotEmpty(&fusion.Campaign, callback.Campaign)
	assignIfNotEmpty(&fusion.Adgroup, callback.Adgroup)
	assignIfNotEmpty(&fusion.Creative, callback.Creative)
	assignIfNotEmpty(&fusion.ActivityKind, callback.ActivityKind)
	assignIfNotEmpty(&fusion.AttributionType, callback.AttributionType)
	assignIfNotEmpty(&fusion.DeviceIP, callback.DeviceIP)
	assignIfNotEmpty(&fusion.UserAgent, callback.UserAgent)
	assignIfNotEmpty(&fusion.GoogleAdID, callback.GoogleAdID)
	assignIfNotEmpty(&fusion.IDFA, callback.IDFA)
	assignIfNotEmpty(&fusion.IDFV, callback.IDFV)
	if callback.AdAccountID != 0 {
		fusion.AdAccountID = callback.AdAccountID
	}
	if callback.ClickTime != nil {
		fusion.ClickTime = callback.ClickTime
	}
	if callback.InstallTime != nil {
		fusion.InstallTime = callback.InstallTime
	}
	if callback.ReattributedAt != nil {
		fusion.ReattributedAt = callback.ReattributedAt
	}
	if callback.AttributionUpdatedAt != nil {
		fusion.AttributionUpdatedAt = callback.AttributionUpdatedAt
	}
	effectiveAt := adjustCallbackTime(callback)
	fusion.AdjustCreatedAt = &effectiveAt
	fusion.IsOrganic, fusion.Reattributed = callback.IsOrganic, callback.Reattributed
	fusion.IsRedownload = callback.IsRedownload
	recordCallbackReceipt(fusion, callback)
	fusion.UpdatedAt = time.Now()
}

func recordCallbackReceipt(fusion *model.VideoAdjustAttribution, callback *model.VideoAdjustAttribution) {
	fusion.CallbackCount++
	fusion.LastCallbackKey = callback.LastCallbackKey
	fusion.CallbackPayload = callback.CallbackPayload
	fusion.CallbackReceivedAt = callback.CallbackReceivedAt
	fusion.UpdatedAt = time.Now()
}

func fusionUpdates(item *model.VideoAdjustAttribution) map[string]any {
	return map[string]any{
		"user_id": item.UserID, "adjust_adid": item.AdjustADID,
		"app_code":   item.AppCode,
		"channel_id": item.ChannelID, "media_id": item.MediaID,
		"ad_account_id": item.AdAccountID,
		"tracker_token": item.TrackerToken,
		"tracker_name":  item.TrackerName, "app_token": item.AppToken,
		"outdated_tracker": item.OutdatedTracker, "outdated_tracker_name": item.OutdatedTrackerName,
		"network":  item.Network,
		"campaign": item.Campaign, "adgroup": item.Adgroup, "creative": item.Creative,
		"click_label": item.ClickLabel, "cost_type": item.CostType,
		"cost_amount": item.CostAmount, "cost_currency": item.CostCurrency,
		"fb_install_referrer": item.FBInstallReferrer, "google_ad_id": item.GoogleAdID,
		"idfa": item.IDFA, "idfv": item.IDFV,
		"activity_kind": item.ActivityKind, "attribution_type": item.AttributionType,
		"is_organic": item.IsOrganic, "reattributed": item.Reattributed,
		"is_redownload": item.IsRedownload,
		"device_ip":     item.DeviceIP, "user_agent": item.UserAgent,
		"click_time": item.ClickTime, "install_time": item.InstallTime,
		"reattributed_at":        item.ReattributedAt,
		"attribution_updated_at": item.AttributionUpdatedAt,
		"adjust_created_at":      item.AdjustCreatedAt, "match_status": item.MatchStatus,
		"callback_count": item.CallbackCount, "last_callback_key": item.LastCallbackKey,
		"app_payload": item.AppPayload, "callback_payload": item.CallbackPayload,
		"app_reported_at": item.AppReportedAt, "callback_received_at": item.CallbackReceivedAt,
		"fused_at": item.FusedAt, "summary_applied": item.SummaryApplied,
		"updated_at": item.UpdatedAt,
	}
}

func sanitizedAdjustPayload(payload map[string]any) map[string]any {
	result := make(map[string]any, len(payload))
	for key, value := range payload {
		if strings.EqualFold(strings.TrimSpace(key), "callback_token") {
			continue
		}
		result[key] = value
	}
	return result
}

func adjustParam(payload map[string]any, names ...string) string {
	for _, name := range names {
		for key, value := range payload {
			if !strings.EqualFold(strings.TrimSpace(key), name) || value == nil {
				continue
			}
			switch typed := value.(type) {
			case string:
				if value := strings.TrimSpace(typed); value != "" {
					return value
				}
			case json.Number:
				return typed.String()
			case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, bool:
				return fmt.Sprint(typed)
			}
		}
	}
	return ""
}

func adjustBoolParam(payload map[string]any, names ...string) bool {
	value := strings.ToLower(adjustParam(payload, names...))
	return value == "1" || value == "true" || value == "yes" || value == "y"
}

func adjustTimeParam(payload map[string]any, names ...string) (*time.Time, error) {
	value := adjustParam(payload, names...)
	if value == "" {
		return nil, nil
	}
	parsed, err := parseAdjustTime(value)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid %s: %v", ErrAdjustCallbackInvalid, names[0], err)
	}
	return &parsed, nil
}

func parseAdjustTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if number, err := strconv.ParseInt(value, 10, 64); err == nil {
		if number >= 1_000_000_000_000 {
			return time.UnixMilli(number), nil
		}
		return time.Unix(number, 0), nil
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05-0700",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05",
	} {
		var parsed time.Time
		var err error
		if layout == "2006-01-02 15:04:05" {
			parsed, err = time.ParseInLocation(layout, value, time.Local)
		} else {
			parsed, err = time.Parse(layout, value)
		}
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("unsupported timestamp format")
}

func resolveAdjustChannel(trackerToken, trackerName string) string {
	mappings := config.Cfg.Adjust.TrackerChannels
	if channel := strings.TrimSpace(mappings[strings.TrimSpace(trackerToken)]); channel != "" {
		return channel
	}
	for tracker, channel := range mappings {
		if strings.EqualFold(strings.TrimSpace(tracker), strings.TrimSpace(trackerName)) {
			return strings.TrimSpace(channel)
		}
	}
	return ""
}

func attributionChannelCode(item *model.VideoAdjustAttribution) string {
	if item.IsOrganic != 0 {
		return ""
	}
	return resolveAdjustChannel(item.TrackerToken, item.TrackerName)
}

func adjustCallbackTime(item *model.VideoAdjustAttribution) time.Time {
	for _, value := range []*time.Time{
		item.AttributionUpdatedAt, item.ReattributedAt, item.AdjustCreatedAt,
		item.InstallTime, item.ClickTime,
	} {
		if value != nil && !value.IsZero() {
			return *value
		}
	}
	if item.CallbackReceivedAt != nil {
		return *item.CallbackReceivedAt
	}
	return item.UpdatedAt
}

func initialAttributionTime(item *model.VideoAdjustAttribution) time.Time {
	for _, value := range []*time.Time{
		item.InstallTime, item.AdjustCreatedAt, item.ClickTime, item.AttributionUpdatedAt,
	} {
		if value != nil && !value.IsZero() {
			return *value
		}
	}
	if item.CallbackReceivedAt != nil {
		return *item.CallbackReceivedAt
	}
	return item.UpdatedAt
}

func classifyAdjustCallback(item *model.VideoAdjustAttribution) adjustCallbackClass {
	kind := normalizeAdjustActivityKind(item.ActivityKind)
	attributionType := normalizeAdjustActivityKind(item.AttributionType)
	switch {
	case kind == "gdpr_forget_device" || attributionType == "gdpr_forget_device":
		return adjustCallbackGDPRForget
	case strings.HasPrefix(kind, "rejected_") || strings.HasPrefix(attributionType, "rejected_"):
		return adjustCallbackIgnored
	case kind == "install_update" || (kind == "attribution" && attributionType == "install_update"):
		return adjustCallbackInstallUpdate
	case strings.HasPrefix(kind, "reattribution") || strings.HasPrefix(attributionType, "reattribution") ||
		item.Reattributed != 0:
		return adjustCallbackReattribution
	case kind == "install":
		return adjustCallbackInitialInstall
	case kind == "attribution" && (attributionType == "" || attributionType == "install" ||
		attributionType == "attribution" || attributionType == "organic"):
		return adjustCallbackInitialInstall
	default:
		return adjustCallbackIgnored
	}
}

func normalizeAdjustActivityKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.NewReplacer("-", "_", " ", "_").Replace(value)
}

func secureTokenEqual(provided, expected string) bool {
	providedHash := sha256.Sum256([]byte(strings.TrimSpace(provided)))
	expectedHash := sha256.Sum256([]byte(strings.TrimSpace(expected)))
	return subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) == 1 && strings.TrimSpace(expected) != ""
}

func assignIfNotEmpty(target *string, value string) {
	if strings.TrimSpace(value) != "" {
		*target = value
	}
}

func boolUint8(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

func getNetworkAccountID(payload map[string]any, network string) uint64 {
	if accountID := adjustAccountID(payload, "ad_account_id"); accountID != 0 {
		return accountID
	}

	network = strings.ToLower(strings.TrimSpace(network))
	var parameterNames []string
	switch {
	case strings.Contains(network, "tiktok"):
		parameterNames = []string{"tiktok_advertiser_id"}
	case strings.Contains(network, "facebook"), strings.Contains(network, "meta"):
		parameterNames = []string{
			"fb_deeplink_account_id",
			"fb_install_referrer_account_id",
			"meta_install_referrer_account_id",
		}
	case strings.Contains(network, "google"):
		parameterNames = []string{"google_ads_external_customer_id"}
	case strings.Contains(network, "apple"):
		parameterNames = []string{"iad_org_id"}
	default:
		parameterNames = []string{
			"tiktok_advertiser_id",
			"fb_deeplink_account_id",
			"fb_install_referrer_account_id",
			"meta_install_referrer_account_id",
			"google_ads_external_customer_id",
			"iad_org_id",
		}
	}
	return adjustAccountID(payload, parameterNames...)
}

func adjustAccountID(payload map[string]any, names ...string) uint64 {
	for _, name := range names {
		value := adjustParam(payload, name)
		if value == "" {
			continue
		}
		accountID, err := strconv.ParseUint(value, 10, 64)
		if err == nil && accountID != 0 {
			return accountID
		}
	}
	return 0
}
