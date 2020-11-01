#include "plugin.h"

AudioEffect* createEffectInstance (audioMasterCallback audioMaster)
{
	return new Plugin (audioMaster);
}
