package lexl

import (
	"math"
	"sort"
)

type interval struct {
	first    int
	last     int
	priority int
	data     interface{}
}

type ivpriset []*interval

func (ps ivpriset) Len() int           { return len(ps) }
func (ps ivpriset) Less(i, j int) bool { return ps[i].priority < ps[j].priority }
func (ps ivpriset) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }

type ivleftset ivpriset

func (ls ivleftset) Less(i, j int) bool { return ls[i].first < ls[j].first }


// resolveIntervals() - Remove overlaps in a slice of intervals by preferencing
// 						regions according to an interval parameter.
func resolveIntervals(intervals []*interval) []*interval {
	
	// Priority sorted, working set of intervals to merge into the result.
	work := ivpriset(make([]*interval, len(intervals)))
	copy(work, intervals)
	sort.Stable(work)
	
	// Result set initial state is a single interval that covers all possible inputs.
	// The result set is lower-bound sorted.  This along with the no-overlaps constraint
	// means choosing interval representatives preserves the order.
	res := ivleftset(make([]*interval, 1, len(intervals)+2))
	res[0] = &interval{0, math.MaxInt64, -1, nil}
	
	// Merge the working set intervals one at a time, from high priority number to low
	// (the lower numbers override the higher).
	for i := len(work) - 1; i >= 0; i-- {
		iv := work[i] 						// The value to merge...
		ni := &interval{					// Copy it.
			first: iv.first,
			last: iv.last,
			priority: iv.priority,
			data: iv.data,
		}
		if ni.first < 0 {					// Normalize -1 values in bounds to extremals.
			ni.first = 0
		}
		if ni.last < 0 {
			ni.last = math.MaxInt64
		}
		idx := sort.Search(len(res), func(n int) bool {		// Find the GLB in the result set.
			return res[n].first >= iv.first
		})									
		if idx == len(res) {								// Handle the case where there is no GLB
			last := res[idx-1]								// because the new interval is the largest.
			if ni.last == last.last {		// If the last values are equal,
				last.last = iv.first-1		// just shorten the old last interval and append the new.
				res = append(res, ni)
			} else {
				after := &interval {		// Otherwise we need to split the old interval and insert
					first: ni.last+1,		// the new value between its parts.
					last: last.last,
					priority: last.priority,
					data: last.data,
				}
				last.last = iv.first-1
				res = append(res,ni)
				res = append(res,after)
			}
		} else {
			//fmt.Printf("search result: %d/%d\n", idx, len(res))
			first := res[idx]  				// Make /first/ the GLB in the result order.
			if first.first > iv.first {		// This means the least value in the to-merge interval
				first = res[idx-1]			// lies inside of /first/.
				idx--
			} 
			
			insertIvs := make([]*interval,0,4)		// Temporary that holds the intervals to insert
													// at the current position.
			if first.first < iv.first { 
				insertIvs = append(insertIvs, &interval{	// A prefix of the GLB interval will be
					first: 		first.first,				// preserved.
					last: 		iv.first-1,
					priority: 	first.priority,
					data: 		first.data,
				})
			}
			insertIvs = append(insertIvs, ni)		// Insert the to-merge interval.
			
			last := first							// Scan past any intervals that are completely 
			lastIdx := idx							// covered by the newly inserted one.
			for last.last <= iv.last {
				lastIdx++
				if lastIdx == len(res) {		
					break
				}
				last = res[lastIdx]
			}
			// Now we know that the last point in the interval to be inserted lies
			// within the interval /last/.  If some section of /last/ is not fully 
			// covered, add that section to the insert slice.
			if iv.last < last.last {
				insertIvs = append(insertIvs, &interval{
					first:		iv.last+1,
					last:		last.last,
					priority:	last.priority,
					data: 		last.data,
				})
			}
			// We can now replace the indexes /idx/ through /lastIdx/, inclusive,
			// with the contents of the temporary.  The ordering is preserved.
			// fmt.Printf("replacing %d-%d with %d new intervals\n", idx, lastIdx, len(insertIvs))
			if lastIdx < len(res)-1 {
				insertIvs = append(insertIvs, res[lastIdx+1:]...)
			}
			res = append(res[0:idx], insertIvs...) 
		}
		//fmt.Println("--- add iteration")
		//for i, r := range res {
		//	var name string
		//	if str, ok := r.data.(string); ok {
		//		name = str
		//	} else {
		//		name = "*"
		//	}
		//	fmt.Printf("%d: {%s: %d - %d (%d)}\n", i, name, r.first, r.last, r.priority)
		//}
	}
	// All of the intervals are now added to the /res/ slice, and none overlap.
	if res[len(res)-1].last == math.MaxInt64 {
		res[len(res)-1].last = -1
	}
	return res
}
/*

	0	.....							N-1
	0 1 2... a=idx ....					N-1
	0 1 2....a a+1 .... b=lastIdx .... 	N-1

	0i 1i 2i .... ki = len(in)
	
		

*/