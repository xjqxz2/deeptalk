package chat

import "testing"

// TestClaimSites 校验单次回答的网页预算按多轮累计扣减。
func TestClaimSites(t *testing.T) {
	urls := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "https://example.com"
		}
		return out
	}

	// 第一轮 5 个，余额 20
	got, left := claimSites(urls(5), 25)
	if len(got) != 5 || left != 20 {
		t.Fatalf("first round = %d/%d, want 5/20", len(got), left)
	}
	// 一轮要 30 个，只给剩下的 20 个
	got, left = claimSites(urls(30), left)
	if len(got) != 20 || left != 0 {
		t.Fatalf("overflow round = %d/%d, want 20/0", len(got), left)
	}
	// 额度用完后不再返回 URL
	got, left = claimSites(urls(3), left)
	if len(got) != 0 || left != 0 {
		t.Fatalf("exhausted round = %d/%d, want 0/0", len(got), left)
	}
	// 空请求不动余额
	if got, left := claimSites(nil, 10); len(got) != 0 || left != 10 {
		t.Fatalf("empty claim = %d/%d", len(got), left)
	}
}
