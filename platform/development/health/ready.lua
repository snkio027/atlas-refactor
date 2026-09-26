-- A condition from an older generation is never evidence for the current spec.
if obj.metadata == nil or obj.metadata.generation == nil then
  return {status = "Progressing", message = "Resource generation is unavailable"}
end
local function conditions(cs, required)
  for _, wanted in ipairs(required) do
    local found = false
    for _, c in ipairs(cs or {}) do
      if c.type == wanted and c.observedGeneration == obj.metadata.generation then
        if c.status == "False" then
          return {status = "Degraded", message = c.message or c.reason or wanted}
        end
        found = c.status == "True"
      end
    end
    if not found then
      return {status = "Progressing", message = "Waiting for current " .. wanted}
    end
  end
  return {status = "Healthy", message = "Current conditions satisfied"}
end

local function sameRef(actual, expected)
  return actual.name == expected.name and
    (actual.namespace or obj.metadata.namespace) == (expected.namespace or obj.metadata.namespace) and
    (actual.group or "gateway.networking.k8s.io") == (expected.group or "gateway.networking.k8s.io") and
    (actual.kind or "Gateway") == (expected.kind or "Gateway") and
    actual.sectionName == expected.sectionName and actual.port == expected.port
end

local status = obj.status or {}
if obj.kind == "HTTPRoute" or obj.kind == "BackendTrafficPolicy" then
  local refs = obj.spec.parentRefs or obj.spec.targetRefs or {}
  local entries = status.parents or status.ancestors or {}
  if #refs == 0 then return {status = "Progressing", message = "No target references"} end
  for _, ref in ipairs(refs) do
    local matched = nil
    for _, entry in ipairs(entries) do
      if entry.controllerName == "gateway.envoyproxy.io/gatewayclass-controller" and
        sameRef(entry.parentRef or entry.ancestorRef or {}, ref) then
        matched = entry
      end
    end
    if matched == nil then return {status = "Progressing", message = "Waiting for target status"} end
    local required = {"Accepted", "ResolvedRefs"}
    if obj.kind == "BackendTrafficPolicy" then required = {"Accepted"} end
    local h = conditions(matched.conditions, required)
    if h.status ~= "Healthy" then return h end
  end
  return {status = "Healthy", message = "Every target accepted current configuration"}
end

local required = {"Ready"}
if obj.kind == "GatewayClass" then required = {"Accepted"} end
if obj.kind == "Gateway" then required = {"Accepted", "Programmed"} end
local h = conditions(status.conditions, required)
if h.status ~= "Healthy" or obj.kind ~= "Gateway" then return h end
for _, listener in ipairs(obj.spec.listeners or {}) do
  local matched = nil
  for _, entry in ipairs(status.listeners or {}) do
    if entry.name == listener.name then matched = entry end
  end
  if matched == nil then return {status = "Progressing", message = "Waiting for listener " .. listener.name} end
  h = conditions(matched.conditions, {"Accepted", "Programmed", "ResolvedRefs"})
  if h.status ~= "Healthy" then return h end
end
return {status = "Healthy", message = "Gateway and all listeners are ready"}
