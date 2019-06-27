local expire_time = 600 -- in seconds

local mqbasenlen = function(key)
  local len = redis.call('llen', key)
  if len > 0 then
    local el = redis.call('lindex', key, 0)
    local base = cjson.decode(el)['offset']
    return {base, len}
  else
    -- a list should never remain empty if its effective
    -- offset should be greater than 0
    return {0, 0}
  end
end

-- mqnext_offset returns the next available offset for a
-- theoretical window on an endless stream
local mqnext = function(key)
  local base_n_len = mqbasenlen(key)
  local base = base_n_len[1]
  local len = base_n_len[2]
  local offset = base + len

  local payload = cjson.encode({offset = offset, set = false, discarded = false})
  assert(redis.call('rpush', key, payload) == len + 1)
  redis.call('expire', key, expire_time)
  return offset
end

-- mqinsert inserts the value for the provided offset over
-- the window to an already existing element. If the element does
-- not exist, the operation fails. get_next_offset must be called
-- so that a specific offset is provided before it can be used
local mqinsert = function(key, offset, value_type, value)
  local base_n_len = mqbasenlen(key)
  local base = base_n_len[1]
  local len = base_n_len[2]
  local index = offset - base

  assert(index >= 0 and index < len)

  local payload = cjson.encode({offset = tonumber(offset), value = value, value_type = value_type, set = true, discarded = false})
  redis.call('expire', key, expire_time)
  return redis.call('lset', key, index, payload)
end

-- mqretrieve returns a window of elements within the list
-- as a contiguous set of elements that have been set
local mqretrieve = function(key, offset, count)
  if redis.call('exists', key) == 0 then
    return {}
  end

  local base_n_len = mqbasenlen(key)
  local base = base_n_len[1]
  local len = base_n_len[2]
  local start = (offset - base)
  local stop = start + count

  -- make sure that stop and start are a valid range within
  -- the list and that start <= stop
  if stop < 0 then
    stop = 0
  end

  if stop >= len then
    stop = len - 1
  end

  if start < 0  then
    start = 0
  end

  if start >= len then
    start = len - 1
  end

  if start > stop then
    stop = start
  end

  redis.call('expire', key, expire_time)
  return redis.call('lrange', key, start, stop)
end

-- mqdiscard discards all elements up to offset if keep_previous is false.
-- It also discards all the elements up to offset + count that have been set.
-- The window cannot be left empty because at least one element is needed
-- to keep track of which is the current window offset.
local mqdiscard = function(key, offset, count, keep_previous)
  offset = tonumber(offset)
  count = tonumber(count)

  local base_n_len = mqbasenlen(key)
  local base = base_n_len[1]
  local len = base_n_len[2]
  if offset < base then
    offset = base
  end
  local start = offset - base
  local stop = len

  if len == 0 then
    -- nothing to discard in that case
    return "OK"
  end

  if not keep_previous then
    -- make sure that we do not delete the last element in the window.
    -- This element is needed to keep context of what's the current
    -- window offset. A window should never be empty
    if start >= len then
      start = len - 1
    end

    assert(start >= 0)

    -- remove all contiguous elements
    redis.call('ltrim', key, start, stop)
    local len = redis.call('llen', key)

    -- check if there are contiguous elements next to stop
    -- that are discarded to extend the range of the trim.
    -- also, len > 1 so that the last element is not removed
    local discarded = true
    while discarded and len > 1 do
      local el = redis.call('lindex', key, 0)
      discarded = cjson.decode(el)['discarded']
      if discarded then
        redis.call('lpop', key)
        len = len - 1
      end
    end

    if count > 0 then
      return mqdiscard(key, offset, count, true)
    end

    return "OK"
  end

  if base == offset then
    return mqdiscard(key, offset + count, 0, false)
  end

  -- mark as discarded all the elements that cannot be discarded
  -- by simply sliding the window
  if count > 0 then
    local els = redis.call('lrange', key, start, start + count - 1)
    for index, el in pairs(els) do
      local decoded = cjson.decode(el)
      decoded['discarded'] = true
      local encoded = cjson.encode(decoded)
      redis.call('lset', key, decoded['offset'] - base, encoded)
    end
  end

  redis.call('expire', key, expire_time)
  return "OK"
end

-- remove the key and all associated resources
local mqremove = function(key)
  return redis.call('del', key)
end

-- attach the API to the global namespace so that it can be
-- accessed from other scripts
rawset(_G, "mqremove", mqremove)
rawset(_G, "mqdiscard", mqdiscard)
rawset(_G, "mqretrieve", mqretrieve)
rawset(_G, "mqinsert", mqinsert)
rawset(_G, "mqnext", mqnext)

-- test the basic functionality of the script
local test = function()
  redis.call('flushall')

  for i = 0, 10  do
    assert(mqnext('example') == i)
    mqinsert('example', i, 'test', cjson.encode({data = i}))
  end

  local t = mqretrieve('example', 0, 10)
  assert(table.getn(t) == 11)
  for i = 0, 10  do
    assert(cjson.decode(t[i+1])['offset'] == i)
  end

  mqdiscard('example', 2, 0, false)
  local t = mqretrieve('example', 0, 10)
  for i = 0, 8  do
    assert(cjson.decode(t[i+1])['offset'] == i + 2)
  end

  mqdiscard('example', 3, 1, true)
  local t = mqretrieve('example', 0, 10)
  assert(table.getn(t) == 9)
  assert(cjson.decode(t[1])['offset'] == 2)
  assert(cjson.decode(t[1])['discarded'] == false)
  assert(cjson.decode(t[2])['offset'] == 3)
  assert(cjson.decode(t[2])['discarded'] == true)
  assert(cjson.decode(t[3])['offset'] == 4)
  assert(cjson.decode(t[3])['discarded'] == false)

  mqdiscard('example', 2, 1, true)
  local t = mqretrieve('example', 0, 10)

  assert(table.getn(t) == 7)
  for i = 0, 6  do
    assert(cjson.decode(t[i+1])['offset'] == i + 4)
  end

  mqdiscard('example', 0, 10, true)
  local t = mqretrieve('example', 0, 10)
  assert(table.getn(t) == 1)

  local ttl = redis.call('ttl', 'example')
  assert(ttl <= 600 and ttl > 100)

  mqremove('example')
  assert(redis.call('exists', 'example') == 0)
end

if ARGV[1] == "test" then
  test()
end
