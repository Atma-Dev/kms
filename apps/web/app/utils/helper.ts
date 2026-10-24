import dayjs from 'dayjs'
import duration from 'dayjs/plugin/duration'
import customParseFormat from 'dayjs/plugin/customParseFormat'
import type { FetchResponse } from 'ofetch'
import type { PayloadDataCollectionPaginated, ResponseError } from '~/types/response'

dayjs.extend(duration)
dayjs.extend(customParseFormat)

/**
 * The API renders most timestamps with `utils.FormatDateTimeVal` —
 * `DD/MM/YYYY HH:mm:ss` — which `new Date()` and dayjs's default parser both
 * read as Invalid Date (month 21). A few resources (queue) send RFC3339
 * instead, so try the API format first and fall back to dayjs's own parsing.
 */
const API_DATETIME_FORMAT = 'DD/MM/YYYY HH:mm:ss'

const parseApiDate = (time?: string | null) => {
  const strict = dayjs(time ?? '', API_DATETIME_FORMAT, true)
  return strict.isValid() ? strict : dayjs(time ?? '')
}

const cutText = (text: string, length: number) => {
  if (text.split(' ').length > 1) {
    const string = text.substring(0, length)
    const splitText = string.split(' ')
    splitText.pop()
    return splitText.join(' ') + '...'
  } else {
    return text
  }
}

const formatDate = (date: string, format: string) => {
  return parseApiDate(date).format(format)
}

const capitalizeFirstLetter = (string: string) => {
  if (string) {
    return string.charAt(0).toUpperCase() + string.slice(1)
  } else {
    return ''
  }
}

const onlyNumber = (string: string) => {
  if (string) {
    return string.replace(/\D/g, '')
  } else {
    return ''
  }
}

const formatCurrency = (number: number) => {
  if (number) {
    const formattedNumber = number.toString().replace(/\D/g, '')
    const rest = formattedNumber.length % 3
    let currency = formattedNumber.substr(0, rest)
    const thousand = formattedNumber.substr(rest).match(/\d{3}/g)
    let separator

    if (thousand) {
      separator = rest ? '.' : ''
      currency += separator + thousand.join('.')
    }

    return currency
  } else {
    return ''
  }
}

const timeAgo = (time: string) => {
  const date = parseApiDate(time)
  if (!date.isValid()) {
    return ''
  }

  const diff = (Date.now() - date.valueOf()) / 1000
  const dayDiff = Math.floor(diff / 86400)

  if (dayDiff < 0 || dayDiff >= 31) {
    return date.format('MMMM DD, YYYY')
  }

  return (
    (dayDiff === 0 &&
      ((diff < 60 && 'just now') ||
        (diff < 120 && '1 minute ago') ||
        (diff < 3600 && Math.floor(diff / 60) + ' minutes ago') ||
        (diff < 7200 && '1 hour ago') ||
        (diff < 86400 && Math.floor(diff / 3600) + ' hours ago'))) ||
    (dayDiff === 1 && 'Yesterday') ||
    (dayDiff < 7 && dayDiff + ' days ago') ||
    (dayDiff < 31 && Math.ceil(dayDiff / 7) + ' weeks ago')
  )
}

const diffTimeByNow = (time: string) => {
  const startDate = dayjs(dayjs().format('YYYY-MM-DD HH:mm:ss').toString())
  const endDate = dayjs(dayjs(time).format('YYYY-MM-DD HH:mm:ss').toString())

  const duration = dayjs.duration(endDate.diff(startDate))
  const milliseconds = Math.floor(duration.asMilliseconds())

  const days = Math.round(milliseconds / 86400000)
  const hours = Math.round((milliseconds % 86400000) / 3600000)
  let minutes = Math.round(((milliseconds % 86400000) % 3600000) / 60000)
  const seconds = Math.round((((milliseconds % 86400000) % 3600000) % 60000) / 1000)

  if (seconds < 30 && seconds >= 0) {
    minutes += 1
  }

  return {
    days: days.toString().length < 2 ? '0' + days : days,
    hours: hours.toString().length < 2 ? '0' + hours : hours,
    minutes: minutes.toString().length < 2 ? '0' + minutes : minutes,
    seconds: seconds.toString().length < 2 ? '0' + seconds : seconds,
  }
}

const isset = (obj: object | string) => {
  if (obj !== null && obj !== undefined) {
    if (typeof obj === 'object' || Array.isArray(obj)) {
      return Object.keys(obj).length
    } else {
      return obj.toString().length
    }
  }

  return false
}
const toSimpleRaw = (obj: object) => {
  return JSON.parse(JSON.stringify(obj))
}

const randomNumbers = (from: number, to: number, length: number) => {
  const numbers = [0]
  for (let i = 1; i < length; i++) {
    numbers.push(Math.ceil(Math.random() * (from - to) + to))
  }

  return numbers
}

const stringToHTML = (arg: string) => {
  const parser = new DOMParser(),
    DOM = parser.parseFromString(arg, 'text/html')
  return DOM.body.childNodes[0] as HTMLElement
}

const slideUp = (el: HTMLElement, duration = 300, callback = (el: HTMLElement) => {}) => {
  el.style.transitionProperty = 'height, margin, padding'
  el.style.transitionDuration = duration + 'ms'
  el.style.height = el.offsetHeight + 'px'
  el.offsetHeight
  el.style.overflow = 'hidden'
  el.style.height = '0'
  el.style.paddingTop = '0'
  el.style.paddingBottom = '0'
  el.style.marginTop = '0'
  el.style.marginBottom = '0'
  window.setTimeout(() => {
    el.style.display = 'none'
    el.style.removeProperty('height')
    el.style.removeProperty('padding-top')
    el.style.removeProperty('padding-bottom')
    el.style.removeProperty('margin-top')
    el.style.removeProperty('margin-bottom')
    el.style.removeProperty('overflow')
    el.style.removeProperty('transition-duration')
    el.style.removeProperty('transition-property')
    callback(el)
  }, duration)
}

const slideDown = (el: HTMLElement, duration = 300, callback = (el: HTMLElement) => {}) => {
  el.style.removeProperty('display')
  let display = window.getComputedStyle(el).display
  if (display === 'none') display = 'block'
  el.style.display = display
  let height = el.offsetHeight
  el.style.overflow = 'hidden'
  el.style.height = '0'
  el.style.paddingTop = '0'
  el.style.paddingBottom = '0'
  el.style.marginTop = '0'
  el.style.marginBottom = '0'
  el.offsetHeight
  el.style.transitionProperty = 'height, margin, padding'
  el.style.transitionDuration = duration + 'ms'
  el.style.height = height + 'px'
  el.style.removeProperty('padding-top')
  el.style.removeProperty('padding-bottom')
  el.style.removeProperty('margin-top')
  el.style.removeProperty('margin-bottom')
  window.setTimeout(() => {
    el.style.removeProperty('height')
    el.style.removeProperty('overflow')
    el.style.removeProperty('transition-duration')
    el.style.removeProperty('transition-property')
    callback(el)
  }, duration)
}

function isFetchResponseError(error: any): error is FetchResponse<ResponseError> {
  return error && typeof error === 'object' && '_data' in error
}

function getDefaultPerPage() {
  return 10
}

function getDefaultPage() {
  return 1
}

function createDefaultPaginated<T>(): PayloadDataCollectionPaginated<T> {
  return {
    meta: {
      current_page: 1,
      first_page_url: '',
      from: 0,
      last_page: 1,
      last_page_url: '',
      links: [],
      next_page_url: null,
      path: '',
      per_page: getDefaultPerPage(),
      prev_page_url: null,
      to: 0,
      total: 0,
    },
    data: [],
  }
}

export {
  createDefaultPaginated,
  getDefaultPerPage,
  getDefaultPage,
  cutText,
  formatDate,
  capitalizeFirstLetter,
  onlyNumber,
  formatCurrency,
  timeAgo,
  diffTimeByNow,
  isset,
  toSimpleRaw,
  randomNumbers,
  stringToHTML,
  slideUp,
  slideDown,
  isFetchResponseError,
}
