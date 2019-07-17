#pragma once

#include <stdio.h>

#define ESTRN 256
typedef char EStr[ESTRN];

#define eprintf(fromat...) snprintf(*estr, ESTRN, fromat)
