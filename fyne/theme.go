//go:build fyne

package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	md3Background   = color.NRGBA{R: 0x1C, G: 0x1B, B: 0x1F, A: 0xFF}
	md3Surface      = color.NRGBA{R: 0x2B, G: 0x29, B: 0x30, A: 0xFF}
	md3SurfaceVar   = color.NRGBA{R: 0x36, G: 0x34, B: 0x3B, A: 0xFF}
	md3Outline      = color.NRGBA{R: 0x45, G: 0x43, B: 0x49, A: 0xFF}
	md3OnSurface    = color.NRGBA{R: 0xE6, G: 0xE1, B: 0xE5, A: 0xFF}
	md3OnSurfaceVar = color.NRGBA{R: 0xC4, G: 0xC0, B: 0xCA, A: 0xFF}
	md3Primary      = color.NRGBA{R: 0xD0, G: 0xBC, B: 0xFF, A: 0xFF}
	md3Error        = color.NRGBA{R: 0xF2, G: 0xB8, B: 0xB5, A: 0xFF}
	md3Success      = color.NRGBA{R: 0xA8, G: 0xDA, B: 0xB5, A: 0xFF}
	md3Info         = color.NRGBA{R: 0x93, G: 0xC5, B: 0xFD, A: 0xFF}
	md3Warn         = color.NRGBA{R: 0xFF, G: 0xD6, B: 0x99, A: 0xFF}
)

type md3DarkTheme struct{}

func (m *md3DarkTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return md3Background
	case theme.ColorNameForeground:
		return md3OnSurface
	case theme.ColorNamePrimary:
		return md3Primary
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x79, G: 0x74, B: 0x7E, A: 0xFF}
	case theme.ColorNameInputBackground:
		return md3Surface
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x79, G: 0x74, B: 0x7E, A: 0xFF}
	case theme.ColorNameSeparator:
		return md3Outline
	case theme.ColorNameButton:
		return md3Surface
	case theme.ColorNameDisabledButton:
		return md3SurfaceVar
	default:
		return theme.DefaultTheme().Color(name, fyne.ThemeVariant(0))
	}
}

func (m *md3DarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m *md3DarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *md3DarkTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 18
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInnerPadding:
		return 4
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func logColor(msg string) color.Color {
	switch {
	case containsAny(msg, "成功", "已断开", "恢复"):
		return md3Success
	case containsAny(msg, "失败", "错误", "原因"):
		return md3Error
	case containsAny(msg, "解析", "开始", "连接池", "每秒"):
		return md3Info
	case containsAny(msg, "停止", "正在断开"):
		return md3Warn
	default:
		return md3OnSurfaceVar
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
