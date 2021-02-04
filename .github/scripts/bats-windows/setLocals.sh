#!/bin/bash

set -e

setLocal EnableDelayedExpansion && pushd %LOCALAPPDATA%\\Temp && del /q/f/s .\\* >nul 2>&1 && rmdir /s/q . >nul 2>&1 && popd
setLocal EnableDelayedExpansion && pushd C:\\batstmp && del /q/f/s .\\* >nul 2>&1 && rmdir /s/q . >nul 2>&1 && popd

exit 0
