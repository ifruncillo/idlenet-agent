; IdleNet Agent Installer Script
!define PRODUCT_NAME "IdleNet Agent"
!define PRODUCT_VERSION "1.0.0"
!define PRODUCT_PUBLISHER "IdleNet"
!define PRODUCT_WEB_SITE "https://idlenet-pilot-qi7t.vercel.app"
!define PRODUCT_DIR_REGKEY "Software\Microsoft\Windows\CurrentVersion\App Paths\idlenet.exe"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

; MUI Settings
!include "MUI2.nsh"
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; Welcome page
!insertmacro MUI_PAGE_WELCOME
; License page (optional, you can add your terms)
!insertmacro MUI_PAGE_LICENSE "LICENSE.txt"
; Directory page
!insertmacro MUI_PAGE_DIRECTORY
; Instfiles page
!insertmacro MUI_PAGE_INSTFILES
; Finish page with option to run
!define MUI_FINISHPAGE_RUN "$INSTDIR\idlenet.exe"
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_INSTFILES

; Language files
!insertmacro MUI_LANGUAGE "English"

; Installer info
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "IdleNet-Setup-${PRODUCT_VERSION}.exe"
InstallDir "$PROGRAMFILES\IdleNet"
InstallDirRegKey HKLM "${PRODUCT_DIR_REGKEY}" ""
ShowInstDetails show
ShowUnInstDetails show

Section "MainSection" SEC01
  SetOutPath "$INSTDIR"
  SetOverwrite try
  
  ; Copy the main executable
  File "idlenet.exe"
  
  ; Create shortcuts
  CreateDirectory "$SMPROGRAMS\IdleNet"
  CreateShortcut "$SMPROGRAMS\IdleNet\IdleNet Agent.lnk" "$INSTDIR\idlenet.exe"
  CreateShortcut "$DESKTOP\IdleNet Agent.lnk" "$INSTDIR\idlenet.exe"
  
  ; Add to Windows startup (optional - ask user first)
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "IdleNet" "$INSTDIR\idlenet.exe"
SectionEnd

Section -AdditionalIcons
  CreateShortcut "$SMPROGRAMS\IdleNet\Uninstall.lnk" "$INSTDIR\uninst.exe"
SectionEnd

Section -Post
  WriteUninstaller "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_DIR_REGKEY}" "" "$INSTDIR\idlenet.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "$(^Name)"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\idlenet.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
SectionEnd

Section Uninstall
  ; Remove from startup
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "IdleNet"
  
  ; Delete files and folders
  Delete "$INSTDIR\uninst.exe"
  Delete "$INSTDIR\idlenet.exe"
  Delete "$SMPROGRAMS\IdleNet\*.*"
  Delete "$DESKTOP\IdleNet Agent.lnk"
  
  RMDir "$SMPROGRAMS\IdleNet"
  RMDir "$INSTDIR"
  
  ; Clean registry
  DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
  DeleteRegKey HKLM "${PRODUCT_DIR_REGKEY}"
  
  SetAutoClose true
SectionEnd