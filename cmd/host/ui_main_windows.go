package main

import (
	"fmt"

	"github.com/JamesHovious/w32"
	"pipelined.dev/audio/vst2"
)

func uiMain(plugin *vst2.Plugin) {
	// Set channels information.
	plugin.SetSpeakerArrangement(
		&vst2.SpeakerArrangement{
			Type:        vst2.SpeakerArrMono,
			NumChannels: int32(2),
		},
		&vst2.SpeakerArrangement{
			Type:        vst2.SpeakerArrMono,
			NumChannels: int32(2),
		},
	)
	// Set buffer size.
	plugin.SetBufferSize(1024)

	fmt.Println("Will start")

	plugin.Start()

	fmt.Println("Started")

	// Get plugin window size
	/*var rect *eRect
	plugin.Dispatch(vst2.EffEditGetRect, 0, 0, vst2.Ptr(&rect), 0)*/
	// FIXME: take window borders into account for sizing

	// Create plugin window and pass it to plugin
	/*hWnd := newNativeWindow(pp, int(rect.right-rect.left)+20, int(rect.bottom-rect.top)+42)
	plugin.Dispatch(vst2.EffEditOpen, 0, 0, hWnd, 0)*/

	// Run event loop on plugin window until it is closed
	var msg w32.MSG
	for {
		if w32.GetMessage(&msg, 0, 0, 0) == 0 {
			break
		}
		w32.TranslateMessage(&msg)
		w32.DispatchMessage(&msg)
	}
	fmt.Println("ret:", int(msg.WParam))
}
