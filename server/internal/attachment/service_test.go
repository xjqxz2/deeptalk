package attachment

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func testService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(db, t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return svc, db
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 24))
	for x := 0; x < 32; x++ {
		for y := 0; y < 24; y++ {
			img.Set(x, y, color.RGBA{R: 77, G: 107, B: 254, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSaveLoadAttach(t *testing.T) {
	svc, _ := testService(t)
	asset, err := svc.Save(context.Background(), 7, "../evil\nname.png", pngBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	if asset.Info.Width != 32 || asset.Info.Height != 24 {
		t.Fatalf("unexpected size: %+v", asset.Info)
	}
	if asset.Info.URL == "" || asset.Info.Name == "" {
		t.Fatalf("missing info: %+v", asset.Info)
	}
	loaded, err := svc.Load(context.Background(), 7, []uint{asset.Info.ID})
	if err != nil || len(loaded) != 1 {
		t.Fatalf("load: %v %d", err, len(loaded))
	}
	dataURL, err := loaded[0].DataURL()
	if err != nil || len(dataURL) < 20 {
		t.Fatalf("data url: %v", err)
	}
	if _, err := svc.Load(context.Background(), 8, []uint{asset.Info.ID}); err == nil {
		t.Fatal("other user should not load attachment")
	}
	if err := svc.Attach(context.Background(), 7, 99, []uint{asset.Info.ID}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Open(context.Background(), 8, asset.Info.ID); err == nil {
		t.Fatal("other user should not open attachment")
	}
}

func TestRejectNonImage(t *testing.T) {
	svc, _ := testService(t)
	if _, err := svc.Save(context.Background(), 1, "x.txt", []byte("not an image")); err == nil {
		t.Fatal("text file should be rejected")
	}
}
