use std::slice;

#[repr(C)]
pub struct DragonflyRedstoneEdge {
    pub from: u32,
    pub to: u32,
    pub weight: i32,
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn dragonfly_redstone_propagate(
    relayers_ptr: *const u8,
    node_count: usize,
    edges_ptr: *const DragonflyRedstoneEdge,
    edge_count: usize,
    powers_ptr: *mut i32,
) -> i32 {
    if node_count == 0 {
        return 0;
    }
    if relayers_ptr.is_null() || powers_ptr.is_null() || (edge_count > 0 && edges_ptr.is_null()) {
        return -1;
    }

    // SAFETY: The caller provides arrays of the declared lengths for the duration of this call.
    let relayers = unsafe { slice::from_raw_parts(relayers_ptr, node_count) };
    let edges = if edge_count == 0 {
        &[]
    } else {
        // SAFETY: edges_ptr was checked and edge_count is supplied by the caller.
        unsafe { slice::from_raw_parts(edges_ptr, edge_count) }
    };
    // SAFETY: powers_ptr was checked and points to node_count writable i32 values.
    let powers = unsafe { slice::from_raw_parts_mut(powers_ptr, node_count) };

    let mut starts = vec![0usize; node_count + 1];
    let mut previous_from = 0usize;
    for (i, edge) in edges.iter().enumerate() {
        let from = edge.from as usize;
        if from >= node_count || edge.to as usize >= node_count {
            return -2;
        }
        if i > 0 && from < previous_from {
            return -3;
        }
        previous_from = from;
        starts[from + 1] += 1;
    }
    for i in 1..starts.len() {
        starts[i] += starts[i - 1];
    }

    let mut queue = Vec::with_capacity(node_count);
    for (i, power) in powers.iter_mut().enumerate() {
        *power = (*power).clamp(0, 15);
        if *power > 0 {
            queue.push(i);
        }
    }

    let mut head = 0usize;
    while head < queue.len() {
        let from = queue[head];
        head += 1;
        if relayers[from] == 0 {
            continue;
        }
        for edge in &edges[starts[from]..starts[from + 1]] {
            let to = edge.to as usize;
            let propagated = (powers[from] - edge.weight).clamp(0, 15);
            if propagated > powers[to] {
                powers[to] = propagated;
                queue.push(to);
            }
        }
    }
    0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn propagates_signal_over_relayers_and_stops_at_endpoints() {
        let relayers = [1, 1, 0, 0, 1];
        let edges = [
            DragonflyRedstoneEdge {
                from: 0,
                to: 1,
                weight: 1,
            },
            DragonflyRedstoneEdge {
                from: 1,
                to: 2,
                weight: 1,
            },
            DragonflyRedstoneEdge {
                from: 1,
                to: 4,
                weight: 2,
            },
            DragonflyRedstoneEdge {
                from: 2,
                to: 3,
                weight: 0,
            },
            DragonflyRedstoneEdge {
                from: 4,
                to: 0,
                weight: 1,
            },
        ];
        let mut powers = [15, 0, 0, 0, 0];
        let status = unsafe {
            dragonfly_redstone_propagate(
                relayers.as_ptr(),
                relayers.len(),
                edges.as_ptr(),
                edges.len(),
                powers.as_mut_ptr(),
            )
        };
        assert_eq!(status, 0);
        assert_eq!(powers, [15, 14, 13, 0, 12]);
    }

    #[test]
    fn rejects_unsorted_edges() {
        let relayers = [1, 1];
        let edges = [
            DragonflyRedstoneEdge {
                from: 1,
                to: 0,
                weight: 1,
            },
            DragonflyRedstoneEdge {
                from: 0,
                to: 1,
                weight: 1,
            },
        ];
        let mut powers = [15, 0];
        let status = unsafe {
            dragonfly_redstone_propagate(
                relayers.as_ptr(), relayers.len(),
                edges.as_ptr(), edges.len(),
                powers.as_mut_ptr(),
            )
        };
        assert_eq!(status, -3);
    }
}
