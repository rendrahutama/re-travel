import { useEffect } from 'react'

const APP_NAME = 'Re-Travel'

function setMetaTag(selector, attrName, attrValue, content) {
  if (!content) return
  let el = document.querySelector(selector)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute(attrName, attrValue)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

export function usePageMeta({ title, description, image, type = 'website' } = {}) {
  useEffect(() => {
    const fullTitle = title ? `${title} | ${APP_NAME}` : APP_NAME
    document.title = fullTitle

    const twitterCard = image ? 'summary_large_image' : 'summary'

    setMetaTag('meta[name="description"]', 'name', 'description', description)
    setMetaTag('meta[property="og:title"]', 'property', 'og:title', fullTitle)
    setMetaTag('meta[property="og:description"]', 'property', 'og:description', description)
    setMetaTag('meta[property="og:image"]', 'property', 'og:image', image)
    setMetaTag('meta[property="og:url"]', 'property', 'og:url', window.location.href)
    setMetaTag('meta[property="og:type"]', 'property', 'og:type', type)
    setMetaTag('meta[name="twitter:card"]', 'name', 'twitter:card', twitterCard)
    setMetaTag('meta[name="twitter:title"]', 'name', 'twitter:title', fullTitle)
    setMetaTag('meta[name="twitter:description"]', 'name', 'twitter:description', description)
    setMetaTag('meta[name="twitter:image"]', 'name', 'twitter:image', image)

    return () => {
      document.title = APP_NAME
    }
  }, [title, description, image, type])
}
