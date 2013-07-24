package storage

import (
	"github.com/garyburd/redigo/redis"
)

var incSlotsScript = redis.NewScript(1, incSlotsScriptSrc)

const incSlotsScriptSrc = `
if redis.call("exists", keys[1]) == 1 then
  local json = redis.call("get", keys[1])
  local user = cjson.decode(json)
  user["slots_used"] = user["slots_used"] + 1
  json = cjson.encode(user)
  redis.call("set", key, json)
  return user["slots_used"]
else
  return nil
end
`

var decSlotsScript = redis.NewScript(1, incSlotsScriptSrc)

const decSlotsScriptSrc = `
if redis.call("exists", keys[1]) == 1 then
  local json = redis.call("get", keys[1])
  local user = cjson.decode(json)
  if user["slots_used"] > 0
    user["slots_used"] = user["slots_used"] - 1
  end
  json = cjson.encode(user)
  redis.call("set", key, json)
  return user["slots_used"]
else
  return nil
end
`

var activeScript = redis.NewScript(1, decSlotsScriptSrc)

const activeScriptSrc = `
if redis.call("exists", keys[1]) == 1 then
  local json = redis.call("get", keys[1])
  local torrent = cjson.decode(json)
  torrent["active"] = true
  json = cjson.encode(torrent)
  redis.call("set", key, json)
  return user["slots_used"]
else
  return nil
end
`

var rmSeederScript = redis.NewScript(2, rmSeederScriptSrc)

const rmSeederScriptSrc = `
if redis.call("EXISTS", keys[1]) == 1 then
  local json = redis.call("GET", keys[1])
  local torrent = cjson.decode(json)
  table.remove(torrent["seeders"], keys[2])
  json = cjson.encode(torrent)
  redis.call("SET", key, json)
  return 0
else
  return nil
end
`
