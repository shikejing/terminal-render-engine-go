package render

import "testing"

func TestBufferSetWide(t *testing.T) {
	b := NewBuffer(10, 3)
	b.SetString(0, 0, "a中b", Style{})
	if got := b.Get(0, 0).Cluster; got != "a" {
		t.Errorf("(0,0) = %q, want a", got)
	}
	c := b.Get(1, 0)
	if c.Cluster != "中" || c.Width != 2 {
		t.Errorf("(1,0) = %+v, want wide 中", c)
	}
	if got := b.Get(2, 0); got.Width != 0 {
		t.Errorf("(2,0).Width = %d, want 0 (continuation)", got.Width)
	}
	if got := b.Get(3, 0).Cluster; got != "b" {
		t.Errorf("(3,0) = %q, want b", got)
	}
}

func TestBufferWideAtRightEdge(t *testing.T) {
	b := NewBuffer(3, 1)
	b.SetString(0, 0, "ab中", Style{}) // 中 在 x=2 放不下
	if got := b.Get(2, 0).Cluster; got != " " {
		t.Errorf("(2,0) = %q, want blank (wide char clipped)", got)
	}
}

func TestBufferOverwriteWideCleanup(t *testing.T) {
	b := NewBuffer(5, 1)
	b.SetString(0, 0, "中文", Style{})
	// 用窄字符覆盖宽字符主格，延续格应被置空
	b.Set(0, 0, "x", Style{})
	if got := b.Get(1, 0); got.Cluster != " " || got.Width != 1 {
		t.Errorf("(1,0) = %+v, want blank after overwrite", got)
	}
	// 覆盖宽字符的延续格，主格应被置空
	b.Clear()
	b.SetString(0, 0, "中文", Style{})
	b.Set(1, 0, "y", Style{})
	if got := b.Get(0, 0); got.Cluster != " " || got.Width != 1 {
		t.Errorf("(0,0) = %+v, want blank after continuation overwrite", got)
	}
}

func TestBufferDiff(t *testing.T) {
	a := NewBuffer(4, 2)
	b := NewBuffer(4, 2)
	if d := b.Diff(a); len(d) != 0 {
		t.Fatalf("identical buffers diff = %d, want 0", len(d))
	}
	b.Set(2, 1, "x", Style{})
	d := b.Diff(a)
	if len(d) != 1 || d[0] != (Pos{2, 1}) {
		t.Fatalf("diff = %v, want [{2 1}]", d)
	}
	// 宽字符修改产生两个差异格（主格+延续格）
	b2 := NewBuffer(4, 2)
	b2.Set(1, 0, "中", Style{})
	d = b2.Diff(a)
	if len(d) != 2 {
		t.Fatalf("wide char diff = %v, want 2 cells", d)
	}
	// 尺寸不同视为全量差异
	c := NewBuffer(3, 3)
	if d := c.Diff(a); len(d) != 9 {
		t.Fatalf("resized diff = %d, want 9", len(d))
	}
	// nil prev 全量
	if d := a.Diff(nil); len(d) != 8 {
		t.Fatalf("nil prev diff = %d, want 8", len(d))
	}
}

func TestBufferStyleDiff(t *testing.T) {
	a := NewBuffer(2, 1)
	b := NewBuffer(2, 1)
	b.Set(0, 0, " ", Style{Bold: true})
	if d := b.Diff(a); len(d) != 1 {
		t.Fatalf("style-only diff = %d, want 1", len(d))
	}
}
