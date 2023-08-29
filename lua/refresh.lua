key = KEYS[1]
value = ARGV[1]
timeout = ARGV[2]

if redis.call('Get',key) == value then
    if redis.call('Set',key,value,timeout) == "OK" then
        return "OK"
    else
        return "refresh failed"
    end
    return "not hold"
end
