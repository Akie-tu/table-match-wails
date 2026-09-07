package backend

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
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

// 单张转JPG(任意格式 → 白底合成, 修复: 原RGBA断言对NRGBA/NYCbCrA失效致透明变黑底)
func convertOne(src, dst string, quality int) error {
	img, err := decodeImage(src)
	if err != nil {
		return err
	}
	// 统一画到白底: draw.Draw 对 RGBA/NRGBA/NYCbCrA 等所有实现通用,
	// 不透明像素原样覆盖白底, 半透明像素与白底正确混合, JPEG 忽略 alpha 也无黑底
	b := img.Bounds()
	bg := image.NewRGBA(b)
	draw.Draw(bg, b, image.White, image.Point{}, draw.Src)
	draw.Draw(bg, b, img, b.Min, draw.Over)
	img = bg

	// 原子写: 先写临时文件再改名, 防转换中断留下半个jpg
	out, err := os.Create(dst + ".tmp")
	if err != nil {
		return err
	}
	encErr := jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
	closeErr := out.Close()
	if encErr != nil {
		os.Remove(dst + ".tmp")
		return encErr
	}
	if closeErr != nil {
		os.Remove(dst + ".tmp")
		return closeErr
	}
	return os.Rename(dst+".tmp", dst)
}

// 批量转JPG: 保留目录树
func RunImgConvert(srcRoot, outRoot string, quality int) (*ImgConvertResult, error) {
	if quality <= 0 || quality > 100 {
		quality = 92
	}
	res := &ImgConvertResult{}
	absOut, err := filepath.Abs(outRoot)
	if err != nil {
		return nil, err
	}
	absSrc, err := filepath.Abs(srcRoot)
	if err != nil {
		return nil, err
	}
	err = filepath.Walk(absSrc, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// 跳过输出目录(修复: 原逻辑目录项直接return nil, 跳过检查永远不生效 → out/out/out嵌套)
		if info.IsDir() {
			abs, _ := filepath.Abs(path)
			if abs == absOut {
				return filepath.SkipDir
			}
			return nil
		}
		res.Total++
		rel, _ := filepath.Rel(absSrc, path)
		ext := strings.ToLower(filepath.Ext(path))
		if imgExts[ext] {
			base := strings.TrimSuffix(path, filepath.Ext(path))
			dst := filepath.Join(absOut, filepath.Dir(rel), filepath.Base(base)+".jpg")
			os.MkdirAll(filepath.Dir(dst), 0o755)
			if err := convertOne(path, dst, quality); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", rel, err))
			} else {
				res.Converted++
			}
		} else {
			// 非图片: 流式复制(修复: 原ReadFile整读大文件会OOM)
			dst := filepath.Join(absOut, rel)
			os.MkdirAll(filepath.Dir(dst), 0o755)
			if err := copyFile(path, dst); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", rel, err))
			} else {
				res.Copied++
			}
		}
		return nil
	})
	return res, err
}

// copyFile: io.Copy流式复制, 支持任意大小文件
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}
