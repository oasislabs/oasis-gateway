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

  local payload = cjson.encode({offset = offset, set = false})
  assert(redis.call('rpush', key, payload) == len + 1)
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

  local payload = cjson.encode({offset = tonumber(offset), value = value, value_type = value_type, set = true})
  return redis.call('lset', key, index, payload)
end

-- mqretrieve returns a window of elements within the list
-- as a contiguous set of elements that have been set
local mqretrieve = function(key, offset, count)
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

  return redis.call('lrange', key, start, stop)
end

-- mqdiscard discards all elements up to offset. The list
-- cannot be left empty because at least one element is needed
-- to keep track of which is the current window offset
local mqdiscard = function(key, offset)
  local base_n_len = mqbasenlen(key)
  local base = base_n_len[1]
  local len = base_n_len[2]
  local start = offset - base
  local stop = len

  -- make sure that we do not delete the last element in the window. This element
  -- is needed to keep context of what's the current window offset. A window should
  -- never be empty
  if start == len then
    start = len - 1
  end

  assert(start >= 0)

  return redis.call('ltrim', key, start, stop)
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

  assert(mqnext('example') == 0)
  assert(mqnext('example') == 1)
  assert(mqnext('example') == 2)
  assert(mqnext('example') == 3)

  mqinsert('example', 0, 'test', '{"data": "my content0"}')
  mqinsert('example', 1, 'test', '{"data": "my content1"}')
  mqinsert('example', 2, 'test', '{"data": "my content2"}')
  mqinsert('example', 3, 'test', '{"data": "my content3"}')

  local t = mqretrieve('example', 0, 10)
  assert(cjson.decode(t[1])['offset'] == 0)
  assert(cjson.decode(t[2])['offset'] == 1)
  assert(cjson.decode(t[3])['offset'] == 2)
  assert(cjson.decode(t[4])['offset'] == 3)

  mqdiscard('example', 2)

  local t = mqretrieve('example', 0, 10)
  assert(cjson.decode(t[1])['offset'] == 2)
  assert(cjson.decode(t[2])['offset'] == 3)

  mqremove('example')
  assert(redis.call('exists', 'example') == 0)
end

if ARGV[1] == "test" then
  test()
end
