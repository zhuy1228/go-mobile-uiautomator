// Package assets 嵌入 UIAutomator2 运行所需的资源文件。
//
// 通过 go:embed 将 u2.jar 和 APK 编译到 Go 二进制中，
// 使用者无需手动管理资源文件路径。
package assets

import "embed"

// 嵌入 UIAutomator2 运行所需的资源文件
var (
	//go:embed u2.jar
	JarData []byte

	//go:embed app-uiautomator.apk
	ApkData []byte

	//go:embed app-uiautomator-test.apk
	ApkTestData []byte
)

// FS 提供对嵌入文件的文件系统访问
//
//go:embed *.jar *.apk
var FS embed.FS
