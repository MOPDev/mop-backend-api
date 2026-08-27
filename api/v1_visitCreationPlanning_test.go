package api

import "testing"

func TestMergeByGeocode(t *testing.T) {
	in := []visitData{
		{Sagsnr: 1, GeocodingAddress: "Ishøj Østergade 43, 2635 Ishøj", Debtors: []debitorData{{DebitorId: 1}}},
		{Sagsnr: 1, GeocodingAddress: "ishøj østergade 43, 2635 ishøj", Debtors: []debitorData{{DebitorId: 2}}},
		{Sagsnr: 2, GeocodingAddress: "some other place", Debtors: []debitorData{{DebitorId: 3}}},
	}
	out := mergeByGeocode(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(out))
	}
	if len(out[0].Debtors) != 2 {
		t.Fatalf("expected 2 debtors merged, got %d", len(out[0].Debtors))
	}
}
