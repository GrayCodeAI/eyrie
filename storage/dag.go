package storage

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

// DAGNode is a single message in the conversation DAG.
type DAGNode struct {
	ID        string
	ParentID  string
	Role      string
	Content   string
	Model     string
	CreatedAt time.Time
	Metadata  map[string]string
}

// DAG is a conversation stored as a directed acyclic graph.
type DAG struct {
	store     Store
	sessionID string
	headID    string
	seq       int
}

// NewDAG opens or creates a conversation DAG for the given session.
func NewDAG(dbPath string, sessionID string) (*DAG, error) {
	store, err := Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &DAG{store: store, sessionID: sessionID}, nil
}

// NewDAGFromStore creates a DAG using an existing store.
func NewDAGFromStore(store Store, sessionID string) *DAG {
	return &DAG{store: store, sessionID: sessionID}
}

// Append adds a new node as a child of the given parent and advances the head.
func (d *DAG) Append(parentID string, role string, content string) (*DAGNode, error) {
	ctx := context.Background()
	if parentID != "" {
		if _, err := d.store.GetNode(ctx, parentID); err != nil {
			return nil, fmt.Errorf("parent node %q not found", parentID)
		}
	}

	node := &DAGNode{
		ID:        dagID(),
		ParentID:  parentID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		Metadata:  make(map[string]string),
	}

	d.seq++
	if err := d.store.CreateNode(ctx, dagNodeToStorage(node, d.sessionID, d.seq)); err != nil {
		return nil, fmt.Errorf("insert node: %w", err)
	}
	d.headID = node.ID
	return node, nil
}

// Fork creates a new branch from the given node.
func (d *DAG) Fork(nodeID string) (*DAGNode, error) {
	ctx := context.Background()
	src, err := d.store.GetNode(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get fork point: %w", err)
	}

	fork := &DAGNode{
		ID:        dagID(),
		ParentID:  src.ParentID,
		Role:      string(src.NodeType),
		Content:   src.Content,
		Model:     src.Model,
		CreatedAt: time.Now().UTC(),
		Metadata:  map[string]string{"forked_from": nodeID},
	}

	d.seq++
	if err := d.store.CreateNode(ctx, dagNodeToStorage(fork, d.sessionID, d.seq)); err != nil {
		return nil, fmt.Errorf("insert fork node: %w", err)
	}
	d.headID = fork.ID
	return fork, nil
}

// History returns the linear path from root to the given node.
func (d *DAG) History(nodeID string) ([]*DAGNode, error) {
	ctx := context.Background()
	if _, err := d.store.GetNode(ctx, nodeID); err != nil {
		return nil, fmt.Errorf("node %q not found", nodeID)
	}
	ancestors, err := d.store.GetAncestors(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	nodes := make([]*DAGNode, len(ancestors))
	for i, sn := range ancestors {
		nodes[i] = storageToDagNode(sn)
	}
	return nodes, nil
}

// Branches returns all child nodes. If nodeID is empty, returns session roots.
func (d *DAG) Branches(nodeID string) ([]*DAGNode, error) {
	ctx := context.Background()
	if nodeID == "" {
		all, err := d.store.ListRootNodes(ctx)
		if err != nil {
			return nil, err
		}
		var nodes []*DAGNode
		for _, sn := range all {
			if sn.RootID == d.sessionID {
				nodes = append(nodes, storageToDagNode(sn))
			}
		}
		return nodes, nil
	}
	children, err := d.store.GetNodeChildren(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	nodes := make([]*DAGNode, len(children))
	for i, sn := range children {
		nodes[i] = storageToDagNode(sn)
	}
	return nodes, nil
}

// Head returns the most recent node in the current branch.
func (d *DAG) Head() (*DAGNode, error) {
	if d.headID == "" {
		return nil, fmt.Errorf("no head for session %q", d.sessionID)
	}
	ctx := context.Background()
	sn, err := d.store.GetNode(ctx, d.headID)
	if err != nil {
		return nil, err
	}
	return storageToDagNode(sn), nil
}

// SetHead moves the head pointer to a specific node.
func (d *DAG) SetHead(nodeID string) error {
	ctx := context.Background()
	if _, err := d.store.GetNode(ctx, nodeID); err != nil {
		return fmt.Errorf("node %q not found", nodeID)
	}
	d.headID = nodeID
	return nil
}

// Prune removes a node and all its descendants.
func (d *DAG) Prune(nodeID string) error {
	ctx := context.Background()
	sn, err := d.store.GetNode(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("node %q not found", nodeID)
	}
	if d.headID != "" {
		subtree, _ := d.store.GetSubtree(ctx, nodeID)
		for _, n := range subtree {
			if n.ID == d.headID {
				if sn.ParentID != "" {
					d.headID = sn.ParentID
				} else {
					d.headID = ""
				}
				break
			}
		}
	}
	return d.store.DeleteNode(ctx, nodeID)
}

// Close closes the underlying store.
func (d *DAG) Close() error {
	return d.store.Close()
}

func dagNodeToStorage(n *DAGNode, sessionID string, seq int) *Node {
	meta, _ := json.Marshal(n.Metadata)
	nodeType := NodeType(n.Role)
	if nodeType == "tool" {
		nodeType = NodeTypeToolCall
	}
	return &Node{
		ID:        n.ID,
		ParentID:  n.ParentID,
		RootID:    sessionID,
		Sequence:  seq,
		NodeType:  nodeType,
		Content:   n.Content,
		Model:     n.Model,
		CreatedAt: n.CreatedAt,
		Metadata:  meta,
	}
}

func storageToDagNode(sn *Node) *DAGNode {
	meta := make(map[string]string)
	if sn.Metadata != nil {
		json.Unmarshal(sn.Metadata, &meta)
	}
	return &DAGNode{
		ID:        sn.ID,
		ParentID:  sn.ParentID,
		Role:      string(sn.NodeType),
		Content:   sn.Content,
		Model:     sn.Model,
		CreatedAt: sn.CreatedAt,
		Metadata:  meta,
	}
}

func dagID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
