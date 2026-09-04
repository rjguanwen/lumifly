// 广场内容类型展示元信息
export const squareTypes = [
  { value: 'record', label: '日记', color: '#4a6cf7', bg: '#eef2ff', icon: 'EditPen' },
  { value: 'milestone', label: '大事记', color: '#d97706', bg: '#fef3c7', icon: 'Trophy' },
  { value: 'idea', label: '灵感', color: '#9333ea', bg: '#f3e8ff', icon: 'Lightning' },
  { value: 'book', label: '读书', color: '#0e9f6e', bg: '#e7f7f0', icon: 'Collection' },
]

export function squareTypeOf(v) {
  return squareTypes.find((t) => t.value === v) || squareTypes[squareTypes.length - 1]
}
