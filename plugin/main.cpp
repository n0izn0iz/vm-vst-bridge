#include "audioeffect.cpp"
#include "audioeffectx.cpp"
#include "vstplugmain.cpp"

#include "plugin.h"

AudioEffect* createEffectInstance (audioMasterCallback audioMaster)
{
	return new Plugin (audioMaster);
}
