from ev3 import *
print('--- guarded limit-based evidence')
for name, P in (("guard evdiv1", dict(EV='guard')), ("guard evdiv2", dict(EV='guard', EVDIV=2)), ("limit evdiv2", dict(EV='limit', EVDIV=2)), ("as written", {})):
    go(name, P); shock(name, P)
