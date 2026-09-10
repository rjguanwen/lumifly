// 生命历（人生日历）周格子的公共计算辅助。
//
// 为什么需要这个文件：组件里「每格 = 1 周」的网格原来每格都要把全部记录/大事记
// 重扫一遍，复杂度是 O(格数 × 数据量)（80 年即 4160 格 × N 条，每次比较还要走
// dayjs 的 isBefore/isAfter）。这里把每格的时间边界一次算好，之后用二分定位每条
// 数据命中的格子，复杂度降为 O(格数 + 数据量 × log 格数)。只改算法，不改判定口径。
//
// 命中口径必须与改动前逐字一致。原判定是：
//     !d.isBefore(weekStart, 'day') && !d.isAfter(weekEnd, 'day')
// 而 dayjs 带单位的 isBefore/isAfter 是**不对称**的（min.js 中实现为
//     isBefore(t, u) => this.endOf(u) < dayjs(t)
//     isAfter(t, u)  => dayjs(t) < this.startOf(u)
// 即只对 this 一侧做 endOf/startOf，参数一侧取原始毫秒值）。因此本文件把判定式写成
//     !(dEnd < wsRaw) && !(weRaw < dStart)
// 其中 dStart/dEnd 是数据当天 startOf/endOf('day') 的毫秒值，wsRaw/weRaw 是格子
// weekStart/weekEnd 的原始毫秒值。注意保留「取反」形式而不是等价的 dEnd >= wsRaw：
// 两者只在数字间等价，一旦 birthDate 非法使比较值变成 NaN，取反写法的结果与改动前
// 一致（NaN 比较恒为 false，取反后恒为 true），化简后就会反过来。

/**
 * 预计算每个周格子的时间边界与展示用日期串。
 * @param {object} birth 出生日的 dayjs 对象（原样参与 add/format，不做任何规整）
 * @param {number} cellCount 格子总数 = 年数 × 52
 */
export function buildWeekBounds(birth, cellCount) {
  const wsRaw = new Float64Array(cellCount) // weekStart 原始毫秒
  const weRaw = new Float64Array(cellCount) // weekEnd 原始毫秒
  const wsStartDay = new Float64Array(cellCount) // weekStart.startOf('day')
  const weEndDay = new Float64Array(cellCount) // weekEnd.endOf('day')
  const startStr = new Array(cellCount)
  const endStr = new Array(cellCount)
  for (let p = 0; p < cellCount; p++) {
    const weekStart = birth.add(p, 'week')
    const weekEnd = weekStart.add(6, 'day')
    startStr[p] = weekStart.format('YYYY-MM-DD')
    endStr[p] = weekEnd.format('YYYY-MM-DD')
    wsRaw[p] = weekStart.valueOf()
    weRaw[p] = weekEnd.valueOf()
    wsStartDay[p] = weekStart.startOf('day').valueOf()
    weEndDay[p] = weekEnd.endOf('day').valueOf()
  }
  return { wsRaw, weRaw, wsStartDay, weEndDay, startStr, endStr, cellCount }
}

/**
 * 第一个 weekEnd >= dStart 的格子下标，即「区间还没有完全落在该日期之前」的最早格子。
 * 返回 cellCount 表示没有这样的格子。
 * weRaw 随下标严格递增（每格比上一格晚 7 天），故可二分。
 */
export function firstCellReachable(bound, dStart) {
  const arr = bound.weRaw
  let lo = 0
  let hi = arr.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (arr[mid] < dStart) lo = mid + 1
    else hi = mid
  }
  return lo
}

/**
 * 最后一个 weekStart <= dEnd 的格子下标，返回 -1 表示没有这样的格子。
 * wsRaw 随下标严格递增，故可二分。
 */
export function lastCellStartingBefore(bound, dEnd) {
  const arr = bound.wsRaw
  let lo = 0
  let hi = arr.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (arr[mid] <= dEnd) lo = mid + 1
    else hi = mid
  }
  return lo - 1
}
