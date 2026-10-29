param([Parameter(Mandatory=$true)][string]$Path)
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot
$source = Get-Content (Join-Path $repo 'main.go') -Raw
if ($source -notmatch 'versionText\s*=\s*"([0-9.]+)"') { throw 'Application version missing' }
$parts = @($Matches[1].Split('.'))
while ($parts.Count -lt 4) { $parts += '0' }
$version = $parts -join '.'
$info = [System.Diagnostics.FileVersionInfo]::GetVersionInfo((Resolve-Path $Path).Path)
$expected = @{ ProductName='Discord Bridge'; ProductVersion=$version; FileVersion=$version; CompanyName='Burhan Uzer'; OriginalFilename='DiscordBridge.exe' }
foreach ($key in $expected.Keys) {
    if ($info.$key -cne $expected[$key]) { throw "Invalid PE metadata: $key" }
}
Write-Host "PE metadata verified: Discord Bridge $version"
