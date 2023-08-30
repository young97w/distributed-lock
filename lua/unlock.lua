key = KEYS[1]
value = ARGV[1]

if redis.call('Get',key) == value then
    res = redis.call('Del',key)
    if res == 1 then
        return "OK"
    else
        return "Not OK"
    end
else
    return "not hold key"
end