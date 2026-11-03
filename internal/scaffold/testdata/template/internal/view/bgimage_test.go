package view

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// TestFitWidthKeepsSmallImages 已经够小的图不该被缩放（白费力且会引入失真）。
func TestFitWidthKeepsSmallImages(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 320, 200))
	got := fitWidth(src, maxBgWidth)
	if got.Bounds().Dx() != 320 || got.Bounds().Dy() != 200 {
		t.Fatalf("小图不应缩放，实际 %v", got.Bounds().Size())
	}
}

// TestFitWidthScalesDown 大图按比例缩到目标宽度。
func TestFitWidthScalesDown(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 6000, 4000))
	got := fitWidth(src, 2000)
	if got.Bounds().Dx() != 2000 || got.Bounds().Dy() != 1333 {
		t.Fatalf("应缩到 2000x1333，实际 %v", got.Bounds().Size())
	}
}

// TestDownscaleAverages 面积平均：左红右蓝缩一半后左右仍是红/蓝，不串色。
func TestDownscaleAverages(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			c := color.RGBA{R: 255, A: 255}
			if x >= 4 {
				c = color.RGBA{B: 255, A: 255}
			}
			src.SetRGBA(x, y, c)
		}
	}
	dst := downscale(src, 4, 2)
	if got := dst.RGBAAt(0, 0); got.R < 250 || got.B > 5 {
		t.Errorf("左上应为红色，实际 %v", got)
	}
	if got := dst.RGBAAt(3, 1); got.B < 250 || got.R > 5 {
		t.Errorf("右下应为蓝色，实际 %v", got)
	}
}

// TestDownscaleYCbCr2x 覆盖 JPEG 的 4:2:0 专用路径：纯色图缩一半后仍是同色。
func TestDownscaleYCbCr2x(t *testing.T) {
	src := image.NewYCbCr(image.Rect(0, 0, 16, 16), image.YCbCrSubsampleRatio420)
	y, cb, cr := color.RGBToYCbCr(200, 120, 60)
	for i := range src.Y {
		src.Y[i] = y
	}
	for i := range src.Cb {
		src.Cb[i] = cb
		src.Cr[i] = cr
	}
	dst := downscale(src, 8, 8)
	want := color.RGBA{R: 200, G: 120, B: 60, A: 255}
	for _, p := range [][2]int{{0, 0}, {7, 7}, {3, 5}} {
		got := dst.RGBAAt(p[0], p[1])
		if abs8(got.R-want.R) > 2 || abs8(got.G-want.G) > 2 || abs8(got.B-want.B) > 2 {
			t.Errorf("(%d,%d) 期望 %v，实际 %v", p[0], p[1], want, got)
		}
	}
}

// TestDecodeBackgroundPNG 走一遍真实解码入口。
func TestDecodeBackgroundPNG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for i := range src.Pix {
		src.Pix[i] = 0x80
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	bmp, err := decodeBackground(buf.Bytes(), maxBgWidth)
	if err != nil {
		t.Fatal(err)
	}
	if bmp == nil {
		t.Fatal("解码结果为空")
	}
}

// TestDecodeBackgroundRejectsGarbage 坏文件要返回错误，界面据此显示「解码失败」。
func TestDecodeBackgroundRejectsGarbage(t *testing.T) {
	if _, err := decodeBackground([]byte("not an image"), maxBgWidth); err == nil {
		t.Fatal("坏数据应返回错误")
	}
}

func abs8(v uint8) int {
	if v > 127 {
		return 255 - int(v)
	}
	return int(v)
}
