"use client"

import * as React from "react"
import { Toast as ToastPrimitive, type ToastObject } from "@base-ui/react/toast"
import { CircleAlert, X } from "lucide-react"
import { cn } from "cn"

const toastManager = ToastPrimitive.createToastManager()

export function toastError(description?: React.ReactNode, title = "操作失败") {
  toastManager.add({
    type: "error",
    title,
    description,
    priority: "high",
    timeout: 6000,
  })
}

function ToastItem({ toast }: { toast: ToastObject<Record<string, never>> }) {
  const { close } = ToastPrimitive.useToastManager()
  const isError = toast.type === "error"

  return (
    <ToastPrimitive.Root
      data-slot="toast"
      toast={toast}
      swipeDirection={["down", "right"]}
      className={cn(
        "flex items-start gap-2.5 rounded-xl border bg-background p-4 shadow-lg outline-none",
        "animate-in slide-in-from-bottom-2 duration-200 data-ended:animate-out data-ended:slide-out-to-right-2",
        isError &&
          "border-[color-mix(in_oklab,var(--color-destructive)25%,var(--color-border))] bg-[color-mix(in_oklab,var(--color-destructive)8%,var(--color-background))] text-destructive"
      )}
    >
      {isError && <CircleAlert className="mt-0.5 size-4 shrink-0 text-destructive" />}
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        {toast.title && (
          <ToastPrimitive.Title
            data-slot="toast-title"
            className={cn("text-sm font-medium", isError && "text-destructive")}
          >
            {toast.title}
          </ToastPrimitive.Title>
        )}
        {toast.description && (
          <ToastPrimitive.Description
            data-slot="toast-description"
            className={cn("whitespace-pre-wrap break-all text-sm text-muted-foreground", isError && "text-destructive")}
          >
            {toast.description}
          </ToastPrimitive.Description>
        )}
      </div>
      <ToastPrimitive.Close
        data-slot="toast-close"
        onClick={() => close(toast.id)}
        className="shrink-0 rounded p-0.5 text-muted-foreground transition-colors hover:text-foreground"
        title="关闭"
      >
        <X className="size-4" />
      </ToastPrimitive.Close>
    </ToastPrimitive.Root>
  )
}

function ToastList() {
  const { toasts } = ToastPrimitive.useToastManager()
  return toasts.map((toast) => <ToastItem key={toast.id} toast={toast} />)
}

function Toaster({ ...props }: React.ComponentProps<typeof ToastPrimitive.Viewport>) {
  return (
    <ToastPrimitive.Provider data-slot="toast-provider" toastManager={toastManager} timeout={5000} limit={3}>
      <ToastPrimitive.Viewport
        data-slot="toast-viewport"
        className="fixed bottom-4 right-4 z-100 flex w-96 max-w-[calc(100vw-2rem)] flex-col gap-2 outline-none"
        {...props}
      >
        <ToastList />
      </ToastPrimitive.Viewport>
    </ToastPrimitive.Provider>
  )
}

export { Toaster }
