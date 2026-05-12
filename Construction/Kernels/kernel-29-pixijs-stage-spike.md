# Kernel 29 - PixiJS Stage Spike

## Expected Purpose
Test whether PixiJS should become Victory's future stage/worldspace renderer without migrating The Cave yet.

## Delivered Behavior
- First Theater hosts a PixiJS stage proving ground
- fire renders in Pixi
- live cards render, can be created from the stage, and can be dragged/moved through the normal action path
- right-click stage and object menus work for create, inspect, edit, flip, duplicate, move, remove, delete, and lock/unlock paths
- the card editor is a movable inspect overlay, not a permanent panel
- card faces are capped so text stays inside the box
- `first-fire` behaves as a removable stage element instead of an untouchable sentinel
- the stage uses a top-left coordinate frame for placement and drag/drop
- debug readouts expose pointer, selection, movement, and server action status

## Closeout Status
Kernel 29 is effectively complete as a proving-ground spike.

What worked well:
- PixiJS proved useful for the stage graph, layering, hit testing, culling, and live object rendering
- the live action loop now works end to end for create, move, duplicate, remove, delete, edit, lock, and inspect flows
- the stage can be reasoned about with visible debug state instead of guesswork

What remains rough:
- nameplate behavior still needs a proper follow-up pass
- hide/show needs an audience-role validation path before it can be called fully proven
- the First Theater implementation still carries proving-ground rough edges and should be organized in the next kernel

Decision:
- retain PixiJS for the stage spike and follow-up organization work
- do not treat the current First Theater implementation as the final polished venue shell
- use Kernel 31 to clean up the proving-ground UI and straighten the remaining affordances

## Evidence Source
- [first-theater/index.html](/opt/victory/frontend/venues/first-theater/index.html)
- [roadmap.md](/opt/victory/Construction/roadmap.md)
- [current-state.md](/opt/victory/Construction/current-state.md)

## TODO
- Carry the remaining nameplate and audience hide/show validation into Kernel 31.
