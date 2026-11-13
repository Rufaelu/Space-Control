; Space Control installer for Windows (NSIS 3).
; Build the game into build\windows first, then:  makensis installer\windows\SpaceControl.nsi
; installer/build-installers.sh (Docker, on Linux), installer/build-windows.sh
; (MSYS2, on Windows) and the GitHub workflow do both steps.
; Installs per user; no administrator rights needed. The game asks for the
; pilot's name itself the first time it starts.

Unicode true
!include "MUI2.nsh"

!define APP "Space Control"
!define EXE "SpaceControl.exe"
!define REGKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\SpaceControl"

Name "${APP}"
OutFile "..\..\dist\SpaceControl-windows-x86_64-setup.exe"
InstallDir "$LOCALAPPDATA\Programs\SpaceControl"
RequestExecutionLevel user
SetCompressor /SOLID lzma

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\${EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch ${APP}"
!define MUI_FINISHPAGE_TEXT "${APP} is installed.$\r$\n$\r$\nTo play your own music, put MP3, OGG or WAV songs in the game's music folder. Start menu > ${APP} > ${APP} Music Folder opens it."
!define MUI_FINISHPAGE_SHOWREADME "$INSTDIR\music"
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Open the music folder (add your own songs)"
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"


Section "Install"
  SetShellVarContext current
  SetOutPath "$INSTDIR"
  File /r "..\..\build\windows\*.*"

  CreateDirectory "$SMPROGRAMS\${APP}"
  CreateShortcut "$SMPROGRAMS\${APP}\${APP}.lnk" "$INSTDIR\${EXE}"
  CreateShortcut "$SMPROGRAMS\${APP}\${APP} Music Folder.lnk" "$INSTDIR\music"
  CreateShortcut "$SMPROGRAMS\${APP}\Uninstall ${APP}.lnk" "$INSTDIR\Uninstall.exe"
  CreateShortcut "$DESKTOP\${APP}.lnk" "$INSTDIR\${EXE}"

  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "${REGKEY}" "DisplayName" "${APP}"
  WriteRegStr HKCU "${REGKEY}" "DisplayIcon" "$INSTDIR\${EXE}"
  WriteRegStr HKCU "${REGKEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegStr HKCU "${REGKEY}" "InstallLocation" "$INSTDIR"
  WriteRegDWORD HKCU "${REGKEY}" "NoModify" 1
  WriteRegDWORD HKCU "${REGKEY}" "NoRepair" 1
SectionEnd

Section "Uninstall"
  SetShellVarContext current
  Delete "$SMPROGRAMS\${APP}\${APP}.lnk"
  Delete "$SMPROGRAMS\${APP}\${APP} Music Folder.lnk"
  Delete "$SMPROGRAMS\${APP}\Uninstall ${APP}.lnk"
  RMDir "$SMPROGRAMS\${APP}"
  Delete "$DESKTOP\${APP}.lnk"
  ; remove only what the installer put there
  Delete "$INSTDIR\${EXE}"
  Delete "$INSTDIR\README.txt"
  Delete "$INSTDIR\*.dll"
  Delete "$INSTDIR\Uninstall.exe"
  ; the music folder goes only if the player left no songs in it
  Delete "$INSTDIR\music\README.txt"
  RMDir "$INSTDIR\music"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "${REGKEY}"
  ; the name and high score in %AppData%\space-control are kept
SectionEnd
