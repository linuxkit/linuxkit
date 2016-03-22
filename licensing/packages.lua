#!/usr/bin/lua

p = {}
f = io.open("/lib/apk/db/installed")

for line in f:lines() do
	if line == "" then
		print(p.P, p.L, p.c)
		p = {}
	else
		k, v = line:match("(%a):(.*)")
		if k then
			p[k] = v
		end
	end
end
