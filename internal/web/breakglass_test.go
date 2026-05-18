package web

import "testing"

func TestDomainGateAllowsBreakglassButKeepsStudentDenial(t *testing.T) {
	cases := []struct {
		name            string
		email           string
		localBreakglass bool
		want            bool
	}{
		{name: "staff domain", email: "alex@wusd.org", want: true},
		{name: "it subdomain", email: "alex@it.wusd.org", want: true},
		{name: "staff subdomain", email: "alex@staff.wusd.org", want: true},
		{name: "student domain denied", email: "student@stu.wusd.org", want: false},
		{name: "outside domain denied", email: "alex@example.com", want: false},
		{name: "local breakglass exception", email: "emergency-alex", localBreakglass: true, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domainGateAllowsDashboardEmail(tc.email, tc.localBreakglass); got != tc.want {
				t.Fatalf("domainGateAllowsDashboardEmail(%q, %v) = %v, want %v", tc.email, tc.localBreakglass, got, tc.want)
			}
		})
	}
}
