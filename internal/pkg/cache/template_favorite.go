package cache

import (
	"fmt"
	"strconv"
	"time"
)

const templateFavoriteCacheTTL = 600 * time.Second

func templateFavoriteKey(userID uint64) string {
	return fmt.Sprintf("user_favorite_key_%d", userID)
}

func templateFavoriteMember(templateID uint64) string {
	return strconv.FormatUint(templateID, 10)
}

// GetTemplateFavorite checks whether templateID is present in the user's
// favorite Set. A missing member is treated as a cache miss so existing
// database favorites can be lazily added to a newly-created or expired Set.
func GetTemplateFavorite(userID, templateID uint64) (favorited bool, found bool) {
	if store == nil {
		return false, false
	}
	favorited, err := store.SIsMember(templateFavoriteKey(userID), templateFavoriteMember(templateID))
	if err != nil || !favorited {
		return false, false
	}
	return true, true
}

// SetTemplateFavorite adds or removes templateID in the user's favorite Set
// after the database transaction has committed.
func SetTemplateFavorite(userID, templateID uint64, favorited bool) error {
	if store == nil {
		return ErrStoreUnavailable
	}

	key := templateFavoriteKey(userID)
	member := templateFavoriteMember(templateID)
	if favorited {
		if _, err := store.SAdd(key, member); err != nil {
			return err
		}
		return store.Expire(key, templateFavoriteCacheTTL)
	}
	_, err := store.SRem(key, member)
	return err
}

func SAddTemplateFavorite(userID uint64, member []string) error {
	if store == nil {
		return ErrStoreUnavailable
	}
	key := templateFavoriteKey(userID)
	if _, err := store.SAdd(key, member); err != nil {
		return err
	}
	return store.Expire(key, templateFavoriteCacheTTL)
}
