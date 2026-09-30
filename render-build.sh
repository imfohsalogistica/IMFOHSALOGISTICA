#!/bin/sh
set -eu
go build -o server_render server_imfohsa.go
mkdir -p data uploads
cp default_state.json data/default_state.json
cp state.json data/state.json
cp admins.json data/admins.json
cp demo-return-1.jpg demo-return-2.jpg uploads/
