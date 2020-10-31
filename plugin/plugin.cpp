//-------------------------------------------------------------------------------------------------------
// VST Plug-Ins SDK
// Version 2.4		$Date: 2006/11/13 09:08:27 $
//
// Category     : VST 2.x SDK Samples
// Filename     : Plugin.cpp
// Created by   : Steinberg Media Technologies
// Description  : Stereo plugin which applies Gain [-oo, 0dB]
//
// � 2006, Steinberg Media Technologies, All Rights Reserved
//-------------------------------------------------------------------------------------------------------

#include "plugin.h"
#include "govst.h"

AudioEffect* createEffectInstance (audioMasterCallback audioMaster)
{
	return new Plugin (audioMaster);
}

//-------------------------------------------------------------------------------------------------------
Plugin::Plugin (audioMasterCallback audioMaster)
: AudioEffectX (audioMaster, 1, 1)	// 1 program, 1 parameter only
{
	setNumInputs (2);		// stereo in
	setNumOutputs (2);		// stereo out
	setUniqueID ('Gain');	// identify
	canProcessReplacing ();	// supports replacing output
	canDoubleReplacing ();	// supports double precision processing

	fGain = 1.f;			// default to 0 dB
	vst_strncpy (programName, "Default-0", kVstMaxProgNameLen);	// default program name

	NewBridge((GoUintptr)this);
}

//-------------------------------------------------------------------------------------------------------
Plugin::~Plugin ()
{
	CloseBridge((GoUintptr)this);
	// nothing to do here
}

//-------------------------------------------------------------------------------------------------------
void Plugin::setProgramName (char* name)
{
	vst_strncpy (programName, name, kVstMaxProgNameLen);
}

//-----------------------------------------------------------------------------------------
void Plugin::getProgramName (char* name)
{
	vst_strncpy (name, programName, kVstMaxProgNameLen);
}

//-----------------------------------------------------------------------------------------
void Plugin::getParameterName (VstInt32 index, char* label)
{
	vst_strncpy (label, "Gain", kVstMaxParamStrLen);
}

//-----------------------------------------------------------------------------------------
void Plugin::getParameterDisplay (VstInt32 index, char* text)
{
	dB2string (fGain, text, kVstMaxParamStrLen);
}

//-----------------------------------------------------------------------------------------
void Plugin::getParameterLabel (VstInt32 index, char* label)
{
	vst_strncpy (label, "dB", kVstMaxParamStrLen);
}

//------------------------------------------------------------------------
bool Plugin::getEffectName (char* name)
{
	vst_strncpy (name, "Gain", kVstMaxEffectNameLen);
	return true;
}

//------------------------------------------------------------------------
bool Plugin::getProductString (char* text)
{
	vst_strncpy (text, "Gain", kVstMaxProductStrLen);
	return true;
}

//------------------------------------------------------------------------
bool Plugin::getVendorString (char* text)
{
	vst_strncpy (text, "Steinberg Media Technologies", kVstMaxVendorStrLen);
	return true;
}

//-----------------------------------------------------------------------------------------
VstInt32 Plugin::getVendorVersion ()
{
	return 1000;
}



// bridged functions below

float Plugin::getParameter (VstInt32 index) {
	return GetParameter((GoUintptr)this, index);
}

void Plugin::setParameter (VstInt32 index, float value) {
	SetParameter((GoUintptr)this, index, value);
}

void Plugin::processReplacing (float** inputs, float** outputs, VstInt32 sampleFrames) {
    ProcessReplacing((GoUintptr)this, (GoFloat32**)inputs, (GoFloat32**)outputs, (GoInt32)sampleFrames);
}

void Plugin::processDoubleReplacing (double** inputs, double** outputs, VstInt32 sampleFrames) {
    ProcessDoubleReplacing((GoUintptr)this, (GoFloat64**)inputs, (GoFloat64**)outputs, (GoInt32)sampleFrames);
}
