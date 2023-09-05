if redis.call('Get',KEYS[1]) == ARGV[1] then
    return redis.call('Del',KEYS[1])
else
    return 0
end