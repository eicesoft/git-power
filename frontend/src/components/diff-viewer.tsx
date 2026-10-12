import { useMemo, type CSSProperties } from 'react'

type DiffLine = {
  oldNumber: number | null
  newNumber: number | null
  kind: 'add' | 'remove' | 'context' | 'hunk' | 'meta'
  text: string
}

function parseDiff(diff: string): DiffLine[] {
  let oldNumber = 0
  let newNumber = 0
  const sourceLines = diff.split(/\r?\n/)
  if (sourceLines.at(-1) === '') sourceLines.pop()

  const visibleLines = sourceLines.filter(
    (line) =>
      !line.startsWith('diff ') &&
      !line.startsWith('index ') &&
      !line.startsWith('--- ') &&
      !line.startsWith('+++ '),
  )

  return visibleLines.map((line) => {
    const hunk = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(line)
    if (hunk) {
      oldNumber = Number(hunk[1])
      newNumber = Number(hunk[2])
      return { oldNumber: null, newNumber: null, kind: 'hunk', text: line }
    }

    if (
      line.startsWith('new file') ||
      line.startsWith('deleted file') ||
      line.startsWith('rename ') ||
      line.startsWith('similarity ') ||
      line.startsWith('Binary ') ||
      line.startsWith('\\')
    ) {
      return { oldNumber: null, newNumber: null, kind: 'meta', text: line }
    }

    if (line.startsWith('+')) {
      return { oldNumber: null, newNumber: newNumber++, kind: 'add', text: line.slice(1) }
    }
    if (line.startsWith('-')) {
      return { oldNumber: oldNumber++, newNumber: null, kind: 'remove', text: line.slice(1) }
    }
    if (line.startsWith(' ')) {
      return { oldNumber: oldNumber++, newNumber: newNumber++, kind: 'context', text: line.slice(1) }
    }

    return { oldNumber: null, newNumber: null, kind: 'meta', text: line }
  })
}

function DiffViewer({
  diff,
  fontFamily,
  fontSize,
}: {
  diff: string
  fontFamily?: string
  fontSize?: number
}) {
  const lines = useMemo(() => parseDiff(diff), [diff])

  const fontStyle: CSSProperties = {}
  if (fontFamily?.trim()) fontStyle.fontFamily = fontFamily.trim()
  if (fontSize && fontSize > 0) {
    fontStyle.fontSize = `${fontSize}px`
    fontStyle.lineHeight = `${Math.round(fontSize * (5 / 3))}px`
  }

  return (
    <div className="w-max min-w-full font-mono text-xs leading-5" style={fontStyle}>
      {lines.map((line, index) => (
        <div
          key={index}
          className={`flex min-w-full ${
            line.kind === 'add'
              ? 'bg-green-500/15'
              : line.kind === 'remove'
                ? 'bg-red-500/15'
                : line.kind === 'hunk'
                  ? 'text-muted-foreground'
                  : line.kind === 'meta'
                    ? 'bg-muted/30 text-muted-foreground'
                    : ''
          }`}
        >
          <span className="w-12 shrink-0 select-none border-r px-2 text-right text-muted-foreground/60">
            {line.oldNumber ?? ''}
          </span>
          <span className="w-12 shrink-0 select-none border-r px-2 text-right text-muted-foreground/60">
            {line.newNumber ?? ''}
          </span>
          <span
            className={`w-6 shrink-0 select-none text-center ${
              line.kind === 'add' ? 'text-green-500' : line.kind === 'remove' ? 'text-red-500' : ''
            }`}
          >
            {line.kind === 'add' ? '+' : line.kind === 'remove' ? '-' : ''}
          </span>
          <code className="whitespace-pre pr-4">{line.text}</code>
        </div>
      ))}
    </div>
  )
}

export { DiffViewer }
