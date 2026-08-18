param(
  [Parameter(Position = 0)]
  [ValidateSet('up', 'down', 'restart', 'status', 'logs', 'upgrade', 'desktop', 'uninstall')]
  [string]$Command = 'status'
)

$ErrorActionPreference = 'Stop'
$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ComposeFile = Join-Path $RootDir 'deploy/compose.yaml'
$StateDir = if ($env:AGENT_NOTIFY_STATE_DIR) { $env:AGENT_NOTIFY_STATE_DIR } else { Join-Path $HOME '.agent-notify' }
$HookBinary = if ($env:AGENT_NOTIFY_HOOK_BINARY) { $env:AGENT_NOTIFY_HOOK_BINARY } else { Join-Path $StateDir 'agent-notify.exe' }
$DesktopDir = Join-Path $env:LOCALAPPDATA 'Agent Notify'
$DesktopBinary = Join-Path $DesktopDir 'Agent Notify.exe'

function Invoke-Compose {
  & docker compose -f $ComposeFile @args
  if ($LASTEXITCODE -ne 0) { throw "docker compose failed with exit code $LASTEXITCODE" }
}

function Sync-Token {
  New-Item -ItemType Directory -Force -Path $StateDir | Out-Null
  $tokenPath = Join-Path $StateDir 'bridge.token'
  $tmpPath = "$tokenPath.tmp.$PID"
  & docker compose -f $ComposeFile exec -T control-plane sh -c 'cat /var/lib/agent-notify/bridge.token' | Set-Content -NoNewline -Encoding ascii $tmpPath
  if ($LASTEXITCODE -ne 0) { throw 'could not read bridge token from control-plane' }
  Move-Item -Force $tmpPath $tokenPath
}

function Invoke-Desktop {
  Push-Location (Join-Path $RootDir 'desktop')
  try {
    & bun install --frozen-lockfile
    & bun run typecheck
    & bun run build
    if ($LASTEXITCODE -ne 0) { throw 'desktop frontend build failed' }
  } finally { Pop-Location }

  New-Item -ItemType Directory -Force -Path $DesktopDir, $StateDir | Out-Null
  $env:GOTOOLCHAIN = 'local'
  Push-Location $RootDir
  try {
    & go build -tags production -o $DesktopBinary ./cmd/agent-notify-desktop
    if ($LASTEXITCODE -ne 0) { throw 'desktop Go build failed' }
  & go build -tags production -o $HookBinary ./cmd/agent-notify
    if ($LASTEXITCODE -ne 0) { throw 'hook runtime build failed' }
  } finally { Pop-Location }

  $ilinkDir = Join-Path $StateDir 'wechat-ilink'
  New-Item -ItemType Directory -Force -Path $ilinkDir | Out-Null
  Copy-Item -Recurse -Force (Join-Path $RootDir 'wechat-ilink/src') $ilinkDir
  Copy-Item -Force (Join-Path $RootDir 'wechat-ilink/package.json') $ilinkDir
  Copy-Item -Force (Join-Path $RootDir 'wechat-ilink/bun.lock') $ilinkDir
  Push-Location $ilinkDir
  try { & bun install --frozen-lockfile --production } finally { Pop-Location }

  if (Get-Command pi -ErrorAction SilentlyContinue) {
    & $HookBinary pi install-extension --scope user --binary $HookBinary
  }

  $runKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
  New-Item -Path $runKey -Force | Out-Null
  New-ItemProperty -Path $runKey -Name 'AgentNotifyTray' -Value ('"{0}" tray' -f $DesktopBinary) -PropertyType String -Force | Out-Null
  Start-Process -FilePath $DesktopBinary -ArgumentList '--show'
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker Desktop is required.' }
switch ($Command) {
  'up'        { Invoke-Compose up -d --build --remove-orphans; Sync-Token }
  'down'      { Invoke-Compose down }
  'restart'   { Invoke-Compose restart; Sync-Token }
  'status'    { Invoke-Compose ps }
  'logs'      { Invoke-Compose logs -f --tail=100 }
  'upgrade'   { Invoke-Compose pull --ignore-buildable; Invoke-Compose up -d --build --remove-orphans; Sync-Token }
  'desktop'   { Invoke-Desktop }
  'uninstall' { Invoke-Compose down --volumes --rmi local; Remove-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name 'AgentNotifyTray' -ErrorAction SilentlyContinue }
}
