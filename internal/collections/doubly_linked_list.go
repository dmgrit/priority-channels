package collections

// ListNode represents a node in the doubly linked list
type ListNode[T any] struct {
	value T
	prev  *ListNode[T]
	next  *ListNode[T]
}

func NewListNode[T any](value T) *ListNode[T] {
	return &ListNode[T]{value: value}
}

func (n *ListNode[T]) Value() T {
	return n.value
}

func (n *ListNode[T]) HasNext() bool {
	return n.next != nil
}

// DoublyLinkedList represents the doubly linked list
type DoublyLinkedList[T any] struct {
	head *ListNode[T]
	tail *ListNode[T]
	size int
}

// Append adds a new node with the given value to the end of the list
func (l *DoublyLinkedList[T]) Append(newNode *ListNode[T]) {
	l.size++
	if l.tail == nil {
		l.head = newNode
		l.tail = newNode
		return
	}
	l.tail.next = newNode
	newNode.prev = l.tail
	l.tail = newNode
}

// RemoveNode removes the given node from the list
func (l *DoublyLinkedList[T]) RemoveNode(node *ListNode[T]) {
	if node == nil {
		return
	}
	l.size--
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}
	node.prev = nil
	node.next = nil
}

func (l *DoublyLinkedList[T]) IsEmpty() bool {
	return l.head == nil
}

func (l *DoublyLinkedList[T]) Len() int {
	return l.size
}

func (l *DoublyLinkedList[T]) LastNode() *ListNode[T] {
	return l.tail
}

type ListIterator[T any] struct {
	current *ListNode[T]
}

func (dll *DoublyLinkedList[T]) Iterator() *ListIterator[T] {
	return &ListIterator[T]{current: dll.head}
}

func (it *ListIterator[T]) HasNext() bool {
	return it.current != nil
}

func (it *ListIterator[T]) Next() T {
	if it.current == nil {
		return getZero[T]()
	}
	value := it.current.value
	it.current = it.current.next
	return value
}

func getZero[T any]() T {
	var result T
	return result
}
