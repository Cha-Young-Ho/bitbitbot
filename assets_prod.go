//go:build prod

package main

import "embed"

// 프로덕션 모드: dist 폴더를 embed
//
//go:embed all:frontend/dist
var assets embed.FS
