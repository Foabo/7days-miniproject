package gee

import "strings"

type node struct {
	pattern  string  // 待匹配的路由，例如 /p/:lang
	part     string  // 路由中的一部分，例如 :lang
	children []*node // 子节点，例如 [doc, tutorial, intro]
	isWild   bool    // 是否精确匹配，part 含有 : 或 * 时为true
}

/**
第一个匹配成功的节点，用于插入
part 代表请求路由的一部分，match的child表示匹配的节点，是树的一部分
*/
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		// || 前一个表达式代表精确匹配到了路径，|| 后一个表达式代表模糊匹配，
		// 模糊匹配的意思是，比如 路由的路径是/p/go
		// p可以被精确匹配，匹配go时候进入新的节点node，也就是匹配到p的节点，
		// go匹配n的孩子，如果n的孩子中有一个精确匹配到go或者它是模糊匹配，就可以返回这个孩子节点
		// 然后可以继续递归查找下去
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

/*
	所有匹配成功的节点，用于查找
	part 代表传入的待匹配的路径的一部分，child代表可能匹配的节点，是树的一部分
*/
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	// 查找所有可能匹配的路径
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

/*
	递归查找每一层的节点，如果没有匹配到当前part的节点，则新建一个

	插入是插入这个pattern，因此我们需要构造pattern对应的parts
	然后从根节点("/")开始向下查找并插入pattern，所以每次都是找孩子和插入孩子
	pattern 待匹配的路由路径
	parts	pattern去除'/'的数组
	height	当前节点的高度，即parts的下标，从0开始
*/
func (n *node) insert(pattern string, parts []string, height int) {
	if len(parts) == height {
		// 所给的路由路径最后一层都匹配完了
		// 这里的pattern是全路径
		// 只有在最后一层节点才会赋予pattern
		n.pattern = pattern
		return
	}

	part := parts[height]       // 当前的节点
	child := n.matchChild(part) // 找是否匹配到这个part
	if child == nil {           // 如果child为空，表示part不在这个trie树中，需要构造child
		child = &node{part: part, isWild: part[0] == '*' || part[0] == ':'} // child不加pattern是因为pattern只在最后一层被赋值
		n.children = append(n.children, child)
	}

	child.insert(pattern, parts, height+1)
}

// 从根节点出发向下递归查找路径
func (n *node) search(parts []string, height int) *node {
	// 匹配到了*，匹配失败，或者匹配到了第len(parts)层节点。
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" { // 表示这个不是最后一个结点，而是中间结点
			return nil
		}
		return n
	}
	part := parts[height]
	children := n.matchChildren(part) // 看trie树中是否能匹配这个part

	for _, child := range children {
		// 对当前的part可能匹配多个，只有最终匹配的会被返回result，即匹配到了第len(parts)层节点。否则都是nil
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}

	return nil

}

// 递归遍历child
func (n *node) travel(list *([]*node)) {
	if n.pattern != "" {
		*list = append(*list, n)
	}
	for _, child := range n.children {
		child.travel(list)
	}
}
