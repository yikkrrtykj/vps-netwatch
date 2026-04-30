import { Badge, Button, Card, Flex, Text, TextField } from "@radix-ui/themes";
import { Crosshair, Plus, Radar, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import type { NodeBasicInfo } from "@/contexts/NodeListContext";

type ApiResponse<T> = {
  status: "success" | "error";
  message?: string;
  data?: T;
};

type DiscoveredTarget = {
  target: string;
  type: "icmp" | "tcp" | "http";
  host: string;
  port?: number;
  network?: string;
  rule?: string;
  process?: string;
  chains?: string[];
  count: number;
};

const requestJSON = async <T,>(url: string, body: unknown): Promise<T> => {
  const response = await fetch(url, {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const text = await response.text();
  let parsed: ApiResponse<T> | null = null;
  try {
    parsed = text ? (JSON.parse(text) as ApiResponse<T>) : null;
  } catch {
    parsed = null;
  }
  if (!response.ok || parsed?.status === "error") {
    throw new Error(parsed?.message || text || response.statusText);
  }
  return (parsed?.data || ({} as T)) as T;
};

const TargetWizard = ({ nodes }: { nodes: NodeBasicInfo[] }) => {
  const clientIDs = useMemo(() => nodes.map((node) => node.uuid), [nodes]);
  const [target, setTarget] = useState("");
  const [name, setName] = useState("");
  const [interval, setIntervalValue] = useState("60");
  const [controller, setController] = useState("http://127.0.0.1:9090");
  const [secret, setSecret] = useState("");
  const [discovered, setDiscovered] = useState<DiscoveredTarget[]>([]);
  const [loading, setLoading] = useState(false);
  const [discovering, setDiscovering] = useState(false);

  const addTarget = async (
    value = target,
    type?: DiscoveredTarget["type"],
    displayName = name,
  ) => {
    const trimmed = value.trim();
    if (!trimmed) {
      toast.error("先输入 IP、域名或 IP:端口");
      return;
    }
    setLoading(true);
    try {
      const data = await requestJSON<{
        task_id: number;
        created: boolean;
        type: string;
        target: string;
      }>("/api/admin/vps-netwatch/target", {
        target: trimmed,
        name: displayName.trim(),
        type,
        interval: Number(interval) || 60,
        clients: clientIDs,
      });
      toast.success(
        data.created
          ? `已加入 ${data.type.toUpperCase()} 监控`
          : "监控目标已存在，已补齐服务器",
      );
      setTarget("");
      setName("");
    } catch (err: any) {
      toast.error(err?.message || "加入监控失败，请确认已登录后台");
    } finally {
      setLoading(false);
    }
  };

  const discoverMihomo = async () => {
    setDiscovering(true);
    try {
      const data = await requestJSON<{ targets: DiscoveredTarget[] }>(
        "/api/admin/vps-netwatch/mihomo/discover",
        {
          controller,
          secret,
          limit: 80,
        },
      );
      setDiscovered(data.targets || []);
      toast.success(`发现 ${data.targets?.length || 0} 个连接目标`);
    } catch (err: any) {
      toast.error(err?.message || "读取 mihomo/Clash 连接失败");
    } finally {
      setDiscovering(false);
    }
  };

  return (
    <Card className="mx-4 mb-3 vpsnw-target-wizard">
      <Flex direction="column" gap="3">
        <Flex
          direction={{ initial: "column", md: "row" }}
          gap="2"
          align={{ initial: "stretch", md: "center" }}
        >
          <Text size="2" weight="bold" className="flex items-center gap-2">
            <Crosshair size={16} />
            监测目标向导
          </Text>
          <TextField.Root
            value={target}
            onChange={(event) => setTarget(event.target.value)}
            placeholder="IP / 域名 / IP:端口 / https://..."
            className="flex-1 min-w-52"
          >
            <TextField.Slot>
              <Search size={14} />
            </TextField.Slot>
          </TextField.Root>
          <TextField.Root
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="名称，可空"
            className="md:w-36"
          />
          <TextField.Root
            value={interval}
            type="number"
            min={5}
            onChange={(event) => setIntervalValue(event.target.value)}
            placeholder="间隔"
            className="md:w-24"
          />
          <Button
            onClick={() => addTarget()}
            disabled={loading}
            className="gap-1"
          >
            <Plus size={15} />
            加入监控
          </Button>
        </Flex>

        <details>
          <summary className="cursor-pointer text-sm text-muted-foreground flex items-center gap-2">
            <Radar size={15} />
            从 mihomo / Clash 当前连接发现游戏服务器
          </summary>
          <Flex direction="column" gap="2" className="mt-3">
            <Flex
              direction={{ initial: "column", md: "row" }}
              gap="2"
              align={{ initial: "stretch", md: "center" }}
            >
              <TextField.Root
                value={controller}
                onChange={(event) => setController(event.target.value)}
                placeholder="http://127.0.0.1:9090"
                className="flex-1"
              />
              <TextField.Root
                value={secret}
                onChange={(event) => setSecret(event.target.value)}
                placeholder="Secret，可空"
                className="md:w-48"
              />
              <Button
                variant="soft"
                onClick={discoverMihomo}
                disabled={discovering}
              >
                发现连接
              </Button>
            </Flex>
            {discovered.length > 0 && (
              <Flex gap="2" wrap="wrap">
                {discovered.map((item) => (
                  <Button
                    key={`${item.type}:${item.target}`}
                    size="1"
                    variant="soft"
                    onClick={() =>
                      addTarget(
                        item.target,
                        item.type,
                        `${item.type.toUpperCase()} ${item.target}`,
                      )
                    }
                  >
                    <Badge size="1" variant="soft">
                      {item.type.toUpperCase()}
                    </Badge>
                    {item.target}
                    <Text size="1" color="gray">
                      {item.count}
                    </Text>
                  </Button>
                ))}
              </Flex>
            )}
          </Flex>
        </details>
      </Flex>
    </Card>
  );
};

export default TargetWizard;
