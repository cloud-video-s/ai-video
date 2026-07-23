package service

import "testing"

func validPointsPackagePayload() PointsPackagePayload {
	return PointsPackagePayload{
		ProductID: "premium_credits_plan", Name: "Premium Credits Plan", PackageID: 1,
		Systems: []string{"android"}, UserTypes: []int{1, 2}, ResourceType: "credits",
		Points: 2900000, Currency: "USD", SalePrice: 39.99, ActualRevenue: 27.99,
		OriginalPrice: 299, ButtonText: "Get More Credits", Status: 1,
	}
}

func TestValidatePointsPackageMoney(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*PointsPackagePayload)
		wantErr bool
	}{
		{name: "valid"},
		{name: "negative price", mutate: func(p *PointsPackagePayload) { p.SalePrice = -1 }, wantErr: true},
		{name: "revenue over price", mutate: func(p *PointsPackagePayload) { p.ActualRevenue = 40 }, wantErr: true},
		{name: "original below sale", mutate: func(p *PointsPackagePayload) { p.OriginalPrice = 20 }, wantErr: true},
		{name: "zero original price", mutate: func(p *PointsPackagePayload) { p.OriginalPrice = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := validPointsPackagePayload()
			if tt.mutate != nil {
				tt.mutate(&payload)
			}
			if err := validatePointsPackageMoney(&payload); (err != nil) != tt.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
