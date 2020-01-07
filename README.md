# merkle-tree
Efficient on-the-fly Merkle Tree implementation. This implementation allows generating minimal partial trees that prove 
membership of several leaves at once, without repeating nodes in the proof or including nodes that can be calculated.

## Usage
_TODO_

## How it Works
### Tree Construction
This library constructs a tree sequentially, in-memory, using `O(log(n))` memory (`n` being the number of tree leaves).

Leaves are added to the tree one-by-one using the `Tree.AddLeaf()` method.

The following animation illustrates the way a tree is constructed:

![legend](readme_assets/Tree%20construction%20legend.svg)
![Tree construction](readme_assets/Tree%20construction.gif)

For each node added we do the following (the leaf layer is 0):
- Check if there's a pending node at the current layer in `pending_left_siblings`.
  - If there isn't - add the currently processing node to the current layer in the list.
  - If there is - use it along with the currently processing node to calculate the parent node and add it to the tree 
  (then clear it from the list).

When there is a power-of-two number of leaves and this process is repeated for all leaves and internal nodes 
encountered, the end result in the `pending_left_siblings` list is that all entries are empty, except the last, which 
represents the root of the tree.

Since we process leaves sequentially from left to right, each left sibling (odd-indexed leaf) will end up being added
to `pending_left_siblings` until its right sibling arrives. At this time both siblings are available and their parent
can be calculated. We then go up a layer and repeat the process.

### Proof Construction
_TODO_
