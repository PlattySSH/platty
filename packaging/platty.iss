#ifndef Version
  #define Version "0.1.0-dev"
#endif

#ifndef SourceDir
  #define SourceDir "..\dist\platty"
#endif

[Setup]
AppId={{8E6C3F2A-9C3B-4C55-9B7E-2D1F0B6A1C11}
AppName=Platty
AppVersion={#Version}
AppPublisher=PlattySSH
AppPublisherURL=https://plattyssh.github.io
AppSupportURL=https://github.com/PlattySSH/platty/issues
AppUpdatesURL=https://github.com/PlattySSH/platty/releases
DefaultDirName={localappdata}\Programs\Platty
DisableProgramGroupPage=yes
DisableDirPage=yes
PrivilegesRequired=lowest
OutputBaseFilename=platty-{#Version}-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
SetupIconFile=..\assets\mascot\platty.ico
UninstallDisplayIcon={app}\platty.exe
UninstallDisplayName=Platty
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.17763

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "{#SourceDir}\platty.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\Platty"; Filename: "{app}\platty.exe"
Name: "{autodesktop}\Platty"; Filename: "{app}\platty.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\platty.exe"; Description: "{cm:LaunchProgram,Platty}"; Flags: nowait postinstall skipifsilent
