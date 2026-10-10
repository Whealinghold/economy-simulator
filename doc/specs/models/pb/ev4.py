from ev3 import *
print('--- variants')
for name, P in (("limit evdiv2", dict(EV='limit', EVDIV=2)), ("limit evdiv4", dict(EV='limit', EVDIV=4)), ("centre evdiv2", dict(EVDIV=2))):
    go(name, P); shock(name, P)
