import { useState } from "react"
import { useParams, useNavigate, useSearchParams } from "react-router"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  ArrowLeft,
  Key,
  Loader2,
  Pencil,
  RefreshCw,
} from "lucide-react"
import { api } from "@/lib/api"
import { usePermission } from "@/hooks/use-permission"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { formatDateTime } from "@/lib/utils"
import { ProductSheet, type ProductItem } from "../../components/product-sheet"
import { ConstraintEditor } from "../../components/constraint-editor"
import { PlanTab } from "../../components/plan-tab"

const STATUS_MAP: Record<string, { label: string; variant: "default" | "secondary" | "outline" }> = {
  unpublished: { label: "未发布", variant: "secondary" },
  published: { label: "已发布", variant: "default" },
  archived: { label: "已归档", variant: "outline" },
}

const STATUS_ACTIONS: Record<string, Array<{ status: string; label: string }>> = {
  unpublished: [
    { status: "published", label: "发布" },
    { status: "archived", label: "归档" },
  ],
  published: [
    { status: "unpublished", label: "下架" },
    { status: "archived", label: "归档" },
  ],
  archived: [
    { status: "unpublished", label: "恢复" },
  ],
}

interface ProductDetail {
  id: number
  name: string
  code: string
  description: string
  status: string
  constraintSchema: ConstraintModule[] | null
  planCount: number
  plans: Array<{
    id: number
    productId: number
    name: string
    constraintValues: Record<string, Record<string, unknown>>
    isDefault: boolean
    sortOrder: number
    createdAt: string
    updatedAt: string
  }>
  createdAt: string
  updatedAt: string
}

interface PublicKeyInfo {
  id: number
  version: number
  publicKey: string
  isCurrent: boolean
  createdAt: string
}

export function Component() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)

  const canUpdate = usePermission("license:product:update")
  const canManagePlan = usePermission("license:plan:manage")
  const canManageKey = usePermission("license:key:manage")

  const { data: product, isLoading } = useQuery({
    queryKey: ["license-product", id],
    queryFn: () => api.get<ProductDetail>(`/api/v1/license/products/${id}`),
    enabled: !!id,
  })

  const statusMutation = useMutation({
    mutationFn: (status: string) =>
      api.patch(`/api/v1/license/products/${id}/status`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-product", id] })
      queryClient.invalidateQueries({ queryKey: ["license-products"] })
      toast.success("状态更新成功")
    },
    onError: (err) => toast.error(err.message),
  })

  const { data: publicKey } = useQuery({
    queryKey: ["license-product-key", id],
    queryFn: () => api.get<PublicKeyInfo>(`/api/v1/license/products/${id}/public-key`),
    enabled: !!id,
  })

  const rotateKeyMutation = useMutation({
    mutationFn: () => api.post(`/api/v1/license/products/${id}/rotate-key`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-product-key", id] })
      toast.success("密钥轮转成功")
    },
    onError: (err) => toast.error(err.message),
  })

  const modules = Array.isArray(product?.constraintSchema) ? product.constraintSchema : []
  const hasSchema = modules.length > 0
  const hasPlans = (product?.planCount ?? 0) > 0
  const requestedTab = searchParams.get("tab")
  const activeTab =
    requestedTab === "info" || requestedTab === "schema" || requestedTab === "plans" || requestedTab === "keys"
      ? requestedTab
      : !hasSchema
        ? "schema"
        : !hasPlans
          ? "plans"
          : "info"

  if (isLoading || !product) {
    return (
      <div className="flex min-h-[200px] items-center justify-center text-muted-foreground">
        加载中...
      </div>
    )
  }

  const status = STATUS_MAP[product.status] ?? { label: product.status, variant: "secondary" as const }
  const actions = STATUS_ACTIONS[product.status] ?? []

  function handleTabChange(value: string) {
    const nextParams = new URLSearchParams(searchParams)
    nextParams.set("tab", value)
    setSearchParams(nextParams, { replace: true })
  }

  const editableProduct: ProductItem = {
    id: product.id,
    name: product.name,
    code: product.code,
    description: product.description,
    status: product.status,
    planCount: product.planCount,
    createdAt: product.createdAt,
    updatedAt: product.updatedAt,
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate("/license/products")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold">{product.name}</h2>
            <Badge variant={status.variant}>{status.label}</Badge>
          </div>
          <p className="text-sm text-muted-foreground font-mono">{product.code}</p>
        </div>
        {canUpdate && (
          <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
            <Pencil className="mr-1.5 h-3.5 w-3.5" />
            编辑
          </Button>
        )}
      </div>

      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList className="h-auto w-fit max-w-full flex-wrap justify-start gap-1 rounded-lg bg-muted/50 p-1">
          <TabsTrigger value="info" className="h-8 flex-none px-3 text-xs sm:text-sm">
            基本信息
          </TabsTrigger>
          <TabsTrigger value="schema" className="h-8 flex-none px-3 text-xs sm:text-sm">
            约束定义
          </TabsTrigger>
          <TabsTrigger value="plans" className="h-8 flex-none px-3 text-xs sm:text-sm">
            套餐管理
          </TabsTrigger>
          <TabsTrigger value="keys" className="h-8 flex-none px-3 text-xs sm:text-sm">
            密钥管理
          </TabsTrigger>
        </TabsList>

        <TabsContent value="info" className="space-y-4">
          <div className="rounded-lg border">
            <div className="grid gap-x-6 gap-y-4 px-4 py-4 text-sm sm:grid-cols-2 lg:grid-cols-3">
              <div>
                <p className="text-muted-foreground">名称</p>
                <p className="mt-1 font-medium">{product.name}</p>
              </div>
              <div>
                <p className="text-muted-foreground">编码</p>
                <p className="mt-1 font-mono">{product.code}</p>
              </div>
              <div>
                <p className="text-muted-foreground">状态</p>
                <div className="mt-1">
                  <Badge variant={status.variant}>{status.label}</Badge>
                </div>
              </div>
              <div>
                <p className="text-muted-foreground">授权模块</p>
                <p className="mt-1">{modules.length}</p>
              </div>
              <div>
                <p className="text-muted-foreground">套餐数量</p>
                <p className="mt-1">{product.planCount}</p>
              </div>
              <div>
                <p className="text-muted-foreground">更新时间</p>
                <p className="mt-1">{formatDateTime(product.updatedAt)}</p>
              </div>
              <div className="sm:col-span-2 lg:col-span-3">
                <p className="text-muted-foreground">描述</p>
                <p className="mt-1 leading-6">
                  {product.description || "暂无描述，可在编辑商品时补充。"}
                </p>
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleTabChange(hasSchema ? (hasPlans ? "keys" : "plans") : "schema")}
            >
              {!hasSchema ? "约束定义" : !hasPlans ? "套餐管理" : "密钥管理"}
            </Button>
            {canUpdate && actions.length > 0 && (
              <>
                {actions.map((action) => (
                  <AlertDialog key={action.status}>
                    <AlertDialogTrigger asChild>
                      <Button variant="outline" size="sm">
                        {action.label}
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>确认{action.label}</AlertDialogTitle>
                        <AlertDialogDescription>
                          确定要将商品 &ldquo;{product.name}&rdquo; {action.label}吗？
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => statusMutation.mutate(action.status)}
                          disabled={statusMutation.isPending}
                        >
                          {action.label}
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                ))}
              </>
            )}
          </div>
        </TabsContent>

        <TabsContent value="schema">
          <ConstraintEditor
            productId={product.id}
            schema={product.constraintSchema}
            canEdit={canUpdate}
          />
        </TabsContent>

        <TabsContent value="plans">
          <PlanTab
            productId={product.id}
            plans={product.plans ?? []}
            constraintSchema={product.constraintSchema}
            canManage={canManagePlan}
            onRequestDefineConstraints={() => handleTabChange("schema")}
          />
        </TabsContent>

        <TabsContent value="keys" className="space-y-4">
          <div className="rounded-lg border p-4">
            {publicKey ? (
              <div className="space-y-4 text-sm">
                <div className="flex items-center gap-2">
                  <Key className="h-4 w-4 text-muted-foreground" />
                  <span className="font-medium">当前密钥</span>
                  <Badge variant="secondary">v{publicKey.version}</Badge>
                </div>
                <div>
                  <p className="text-muted-foreground">公钥</p>
                  <pre className="mt-1 rounded bg-muted p-3 text-xs break-all whitespace-pre-wrap font-mono">
                    {publicKey.publicKey}
                  </pre>
                </div>
                <div>
                  <p className="text-muted-foreground">创建时间</p>
                  <p className="mt-1">{formatDateTime(publicKey.createdAt)}</p>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">暂无密钥信息</p>
            )}
          </div>

          {canManageKey && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                  密钥轮转
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>确认密钥轮转</AlertDialogTitle>
                  <AlertDialogDescription>
                    密钥轮转将生成新的密钥对，旧密钥将被标记为已撤销。已使用旧密钥签发的许可证仍可验证。此操作不可撤销。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => rotateKeyMutation.mutate()}
                    disabled={rotateKeyMutation.isPending}
                  >
                    {rotateKeyMutation.isPending ? (
                      <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                    ) : null}
                    确认轮转
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}
        </TabsContent>
      </Tabs>

      <ProductSheet open={editOpen} onOpenChange={setEditOpen} product={editableProduct} />
    </div>
  )
}
