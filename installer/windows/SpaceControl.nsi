; Space Control installer for Windows (NSIS 3).
; Build the game into build\windows first (installer\build-windows.sh or the
; GitHub workflow does both steps), then:  makensis installer\windows\SpaceControl.nsi
; Installs per user; no administrator rights needed.

Unicode true
!include "MUI2.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

!define APP "Space Control"
!define EXE "SpaceControl.exe"
!define REGKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\SpaceControl"

Name "${APP}"
OutFile "..\..\dist\SpaceControl-windows-x86_64-setup.exe"
InstallDir "$LOCALAPPDATA\Programs\SpaceControl"
RequestExecutionLevel user
SetCompressor /SOLID lzma

Var PilotName
Var NameField

!insertmacro MUI_PAGE_WELCOME
Page custom NamePageShow NamePageLeave
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\${EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch ${APP}"
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

; --- page: ask for the pilot's name (the game uses it to congratulate them) ---
Function NamePageShow
  !insertmacro MUI_HEADER_TEXT "Pilot name" "The game will call you by this name."
  nsDialogs::Create 1018
  Pop $0
  ${NSD_CreateLabel} 0 0 100% 24u "What is your name, pilot?$\r$\n(Letters, digits, spaces and dashes; up to 12 characters.)"
  Pop $0
  ${If} $PilotName == ""
    ReadEnvStr $PilotName USERNAME
  ${EndIf}
  ${NSD_CreateText} 0 30u 60% 13u "$PilotName"
  Pop $NameField
  ${NSD_SetTextLimit} $NameField 12
  ${NSD_SetFocus} $NameField
  nsDialogs::Show
FunctionEnd

Function NamePageLeave
  ${NSD_GetText} $NameField $PilotName
  ${If} $PilotName == ""
    MessageBox MB_ICONEXCLAMATION "Please enter a name."
    Abort
  ${EndIf}
FunctionEnd

Section "Install"
  SetShellVarContext current
  SetOutPath "$INSTDIR"
  File /r "..\..\build\windows\*.*"

  ; the game reads %AppData%\space-control\name and tidies the text itself
  CreateDirectory "$APPDATA\space-control"
  FileOpen $0 "$APPDATA\space-control\name" w
  FileWrite $0 "$PilotName$\r$\n"
  FileClose $0

  CreateDirectory "$SMPROGRAMS\${APP}"
  CreateShortcut "$SMPROGRAMS\${APP}\${APP}.lnk" "$INSTDIR\${EXE}"
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
  Delete "$SMPROGRAMS\${APP}\Uninstall ${APP}.lnk"
  RMDir "$SMPROGRAMS\${APP}"
  Delete "$DESKTOP\${APP}.lnk"
  ; remove only what the installer put there
  Delete "$INSTDIR\${EXE}"
  Delete "$INSTDIR\README.txt"
  Delete "$INSTDIR\*.dll"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "${REGKEY}"
  ; the name and high score in %AppData%\space-control are kept
SectionEnd
