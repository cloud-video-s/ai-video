package app

// SeedChannelAdmin reconciles channel-management APIs, menu permissions and
// the super-admin grant for fresh and existing installations.
//func SeedChannelAdmin() error {
//	return DB.Transaction(func(tx *gorm.DB) error {
//		seeds := []templateAPISeed{
//			{Path: "/admin/channels", Method: "GET", Group: "渠道管理", Description: "渠道列表"},
//			{Path: "/admin/channels/:id", Method: "GET", Group: "渠道管理", Description: "渠道详情"},
//			{Path: "/admin/channels", Method: "POST", Group: "渠道管理", Description: "新增渠道"},
//			{Path: "/admin/channels/:id", Method: "PUT", Group: "渠道管理", Description: "编辑渠道"},
//			{Path: "/admin/channels/:id/status", Method: "PATCH", Group: "渠道管理", Description: "切换渠道状态"},
//			{Path: "/admin/channels/:id", Method: "DELETE", Group: "渠道管理", Description: "删除渠道"},
//		}
//
//		apis := make([]model.VideoAPI, 0, len(seeds))
//		for _, seed := range seeds {
//			api, err := upsertTemplateAPI(tx, seed)
//			if err != nil {
//				return err
//			}
//			apis = append(apis, *api)
//		}
//
//		page, err := upsertTemplateMenu(tx, model.VideoMenu{
//			ParentID: 0, Name: "渠道管理", Path: "/channel/list",
//			Component: "channel/list/index", Icon: "Promotion", Sort: 3, Type: 1,
//			Permission: "channel:list", Visible: 1, Status: 1,
//		})
//		if err != nil {
//			return err
//		}
//		if err := tx.Model(page).Association("APIs").Replace(apis[0], apis[1]); err != nil {
//			return err
//		}
//
//		buttonSeeds := []struct {
//			menu model.VideoMenu
//			apis []model.VideoAPI
//		}{
//			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增渠道", Sort: 1, Type: 2, Permission: "channel:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
//			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑渠道", Sort: 2, Type: 2, Permission: "channel:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[3], apis[4]}},
//			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除渠道", Sort: 3, Type: 2, Permission: "channel:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[5]}},
//		}
//		menus := []model.VideoMenu{*page}
//		for _, seed := range buttonSeeds {
//			button, err := upsertTemplateMenu(tx, seed.menu)
//			if err != nil {
//				return err
//			}
//			if err := tx.Model(button).Association("APIs").Replace(seed.apis); err != nil {
//				return err
//			}
//			menus = append(menus, *button)
//		}
//
//		var adminRole model.VideoRole
//		if err := tx.Where("code = ?", domain.SuperAdminRoleCode).First(&adminRole).Error; err != nil {
//			return err
//		}
//		return tx.Model(&adminRole).Association("Menus").Append(menus)
//	})
//}
