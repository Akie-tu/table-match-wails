package backend

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/webp"
)

// 图片转换结果
type ImgConvertResult struct {
	Total     int      `json:"total"`
	Converted int      `json:"converted"`
	Copied    int      `json:"copied"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors"`
}

// 支持转换的格式
var imgExts = map[string]bool{
	".webp": true, ".png": true, ".jpg": true, ".jpeg": true,
	".bmp": true, ".gif": true, ".tiff": true, ".jfif": true,
}

// 解码图片(webp用x/image, 其他用标准库)
func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".webp" {
		return webp.Decode(f)
	}
	img, _, err := image.Decode(f)
	return img, err
}

// 单张转JPG(白底合成RGBA)
func convertOne(src, dst string, quality int) error {
	img, err := decodeImage(src)
	if err != nil {
		return err
	}
	// RGBA/透明 → 白底
	b := img.Bounds()
	if rgba, ok := img.(*image.RGBA); ok && hasAlpha(rgba) {
		bg := image.NewRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				bg.Set(x, y, image.White)
			}
		}
		// 合成
		drawOver(bg, rgba)
		img = bg
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	return jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
}

// 是否有透明像素
func hasAlpha(rgba *image.RGBA) bool {
	for i := 3; i < len(rgba.Pix); i += 4 {
		if rgba.Pix[i] != 255 {
			return true
		}
	}
	return false
}

// 简单合成(alpha混合白底)
func drawOver(dst *image.RGBA, src *image.RGBA) {
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			s := src.RGBAAt(x, y)
			if s.A == 255 {
				dst.SetRGBA(x, y, s)
			} else if s.A > 0 {
				// alpha混合
				dr := dst.RGBAAt(x, y)
				a := float64(s.A) / 255.0
				dr.R = uint8(float64(s.R)*a + float64(dr.R)*(1-a))
				dr.G = uint8(float64(s.G)*a + float64(dr.G)*(1-a))
				dr.B = uint8(float64(s.B)*a + float64(dr.B)*(1-a))
				dst.SetRGBA(x, y, dr)
			}
		}
	}
}

// 批量转JPG: 保留目录树
func RunImgConvert(srcRoot, outRoot string, quality int) (*ImgConvertResult, error) {
	if quality <= 0 || quality > 100 {
		quality = 92
	}
	res := &ImgConvertResult{}
	absOut, _ := filepath.Abs(outRoot)
	err := filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		res.Total++
		rel, _ := filepath.Rel(srcRoot, path)
		ext := strings.ToLower(filepath.Ext(path))
		if imgExts[ext] {
			base := strings.TrimSuffix(path, filepath.Ext(path))
			dst := filepath.Join(outRoot, filepath.Dir(rel), filepath.Base(base)+".jpg")
			os.MkdirAll(filepath.Dir(dst), 0o755)
			if err := convertOne(path, dst, quality); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", rel, err))
			} else {
				res.Converted++
			}
		} else {
			// 非图片: 复制
			dst := filepath.Join(outRoot, rel)
			os.MkdirAll(filepath.Dir(dst), 0o755)
			if err := copyFile(path, dst); err != nil {
				res.Failed++
			} else {
				res.Copied++
			}
		}
		// 跳过输出目录
		abs, _ := filepath.Abs(path)
		if strings.HasPrefix(abs, absOut) && path != srcRoot {
			return filepath.SkipDir
		}
		return nil
	})
	return res, err
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// png编码(备用, 未用)
var _ = png.Encode