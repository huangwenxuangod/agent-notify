package doctor

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
)

// hookBinaryPath 从 agent 的配置文件里取出已注册 hook 命令指向的二进制路径。
// 四个 agent 的配置都是 JSON,只是 hook 数组的嵌套深度不同,所以递归遍历
// 找第一个含 marker 的 command 字符串即可,无需按 agent 分支。
// 找不到返回空串。
func hookBinaryPath(settingsPath, marker string) string {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}
	cmd := findCommandWithMarker(root, marker)
	if cmd == "" {
		return ""
	}
	return common.BinaryPathFromHookCommand(cmd, marker)
}

func findCommandWithMarker(node any, marker string) string {
	switch v := node.(type) {
	case map[string]any:
		if cmd, ok := v["command"].(string); ok && strings.Contains(cmd, marker) {
			return cmd
		}
		for _, child := range v {
			if found := findCommandWithMarker(child, marker); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range v {
			if found := findCommandWithMarker(child, marker); found != "" {
				return found
			}
		}
	}
	return ""
}

// hookBinaryMissing 判断已注册的 hook 命令是否指向一个不存在的二进制。
// 典型场景:用户先用本地构建安装,后改用 bunx,旧路径已被删除(issue #34)。
// 无法判定时(读不到配置、解析不出路径、裸命令名依赖 PATH)返回 false,不误报。
func hookBinaryMissing(settingsPath, marker string) bool {
	binPath := hookBinaryPath(settingsPath, marker)
	if binPath == "" {
		return false
	}
	if !strings.ContainsAny(binPath, `/\`) {
		return false
	}
	_, err := os.Stat(binPath)
	return os.IsNotExist(err)
}

// hookBinaryMissingSource extracts the configured runtime path from source
// based hooks (Hermes Python and OpenClaw JavaScript). They are not JSON hook
// settings, but both write a quoted absolute binary immediately before marker.
func hookBinaryMissingSource(sourcePath, marker string) bool {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return false
	}
	text := string(data)
	markerOffset := strings.Index(text, marker)
	if markerOffset < 0 {
		return false
	}
	beforeMarker := text[:markerOffset]
	var matches []string
	if strings.Contains(text, "subprocess.run") {
		pythonHook := regexp.MustCompile(`(?s)subprocess\.run\(\[\s*["']([^"']+)["']\s*,\s*["']` + regexp.QuoteMeta(marker) + `["']`)
		matches = pythonHook.FindStringSubmatch(text)
	} else {
		jsBinary := regexp.MustCompile(`(?m)^\s*const binary\s*=\s*["']([^"']+)["'];`)
		matches = jsBinary.FindStringSubmatch(beforeMarker)
	}
	if len(matches) != 2 || !strings.ContainsAny(matches[1], `/\`) {
		return false
	}
	_, err = os.Stat(matches[1])
	return os.IsNotExist(err)
}
