import { type MouseEvent } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'

// MarkdownView 渲染 README 等 Markdown 文本，排版样式见 index.css 的 .markdown-body。
// 链接统一交给系统浏览器打开：WebView 内导航会顶掉应用界面，README 里的相对链接也解析不到。
export function MarkdownView({ content }: { content: string }) {
  const openLink = (event: MouseEvent<HTMLDivElement>) => {
    const anchor = (event.target as HTMLElement).closest('a')
    if (!anchor) return
    event.preventDefault()
    const href = anchor.getAttribute('href') ?? ''
    if (/^https?:\/\//i.test(href)) BrowserOpenURL(href)
  }

  return (
    <div className="markdown-body" onClick={openLink}>
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
    </div>
  )
}
