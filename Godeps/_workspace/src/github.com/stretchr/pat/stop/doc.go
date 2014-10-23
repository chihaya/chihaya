// Package stop represents a pattern for types that need to do some work
// when stopping. The StopChan method returns a <-chan stop.Signal which
// is closed when the operation has completed.
//
// Stopper types when implementing the stop channel pattern should use stop.Make
// to create and store a stop channel, and close the channel once stopping has completed:
//     func New() Type {
//       t := new(Type)
//       t.stopChan = stop.Make()
//       return t
//     }
//     func (t Type) Stop() {
//       go func(){
//         // TODO: tear stuff down
//         close(t.stopChan)
//       }()
//     }
//     func (t Type) StopChan() <-chan stop.Signal {
//       return t.stopChan
//     }
//
// Stopper types can be stopped in the following ways:
//     // stop and forget
//     t.Stop(1 * time.Second)
//
//     // stop and wait
//     t.Stop(1 * time.Second)
//     <-t.StopChan()
//
//     // stop, do more work, then wait
//     t.Stop(1 * time.Second);
//     // do more work
//     <-t.StopChan()
//
//     // stop and timeout after 1 second
//     t.Stop(1 * time.Second)
//     select {
//     case <-t.StopChan():
//     case <-time.After(1 * time.Second):
//     }
//
//     // stop.All is the same as calling Stop() then StopChan() so
//     // all above patterns also work on many Stopper types,
//     // for example; stop and wait for many things:
//     <-stop.All(1 * time.Second, t1, t2, t3)
package stop
