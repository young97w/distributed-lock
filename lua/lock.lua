---
--- Generated by EmmyLua(https://github.com/EmmyLua)
--- Created by yangyuan.
--- DateTime: 2023/9/5 22:07

local val = redis.call('get',KEYS[1])
if val == false then
    return redis.call('set',KEYS[1],ARGV[1],'EX',ARGV[2])
elseif val == ARGV[1] then
    return redis.call('expire',KEYS[1],ARGV[2])
else
    return 0
end