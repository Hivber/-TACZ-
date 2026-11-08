#ifndef DRAGONFLY_REDSTONE_H
#define DRAGONFLY_REDSTONE_H

#include <stddef.h>
#include <stdint.h>

typedef struct {
    uint32_t from;
    uint32_t to;
    int32_t weight;
} DragonflyRedstoneEdge;

int32_t dragonfly_redstone_propagate(
    const uint8_t *relayers,
    size_t node_count,
    const DragonflyRedstoneEdge *edges,
    size_t edge_count,
    int32_t *powers
);

#endif
