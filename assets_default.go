//go:build !prod

package main

import "embed"

// 개발/기본 모드: 프론트엔드 dev 서버를 사용하지만
// AssetServer 옵션 유효성 검사를 통과하기 위해 빈 embed.FS를 설정합니다.
var assets embed.FS
