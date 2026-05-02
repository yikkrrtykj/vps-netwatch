import { Badge, Flex, Text } from "@radix-ui/themes";
import { Activity, AlertTriangle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useRPC2Call } from "@/contexts/RPC2Context";

type PingRecord = {
  client: string;
  task_id: number;
  time: string;
  value: number;
};

type TaskInfo = {
  id: number;
  name: string;
  type?: string;
  interval?: number;
  loss?: number;
  min?: number;
  max?: number;
  avg?: number;
  total?: number;
};

type PingResponse = {
  records?: PingRecord[];
  tasks?: TaskInfo[];
};

type Summary = {
  task: TaskInfo;
  latest: number | null;
  avg: number;
  max: number;
  lossRate: number;
  jitter: number;
  anomaly: string | null;
};

const percentile = (values: number[], ratio: number) => {
  if (!values.length) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const index = Math.min(
    sorted.length - 1,
    Math.max(0, Math.ceil(sorted.length * ratio) - 1),
  );
  return sorted[index];
};

const analyzeTask = (task: TaskInfo, records: PingRecord[]): Summary | null => {
  if (!records.length) return null;
  const sorted = [...records].sort(
    (a, b) => new Date(a.time).getTime() - new Date(b.time).getTime(),
  );
  const values = sorted.map((item) => item.value).filter((value) => value >= 0);
  const latestRecord = sorted[sorted.length - 1];
  const latest = latestRecord?.value >= 0 ? latestRecord.value : null;
  const avg = values.length
    ? Math.round(values.reduce((sum, value) => sum + value, 0) / values.length)
    : 0;
  const max = values.length ? Math.max(...values) : 0;
  const p50 = percentile(values, 0.5);
  const p95 = percentile(values, 0.95);
  const jitter = Math.max(0, p95 - p50);
  const lossCount = sorted.length - values.length;
  const measuredLoss = sorted.length ? (lossCount / sorted.length) * 100 : 0;
  const lossRate =
    typeof task.loss === "number" && task.loss > 0 ? task.loss : measuredLoss;

  const peakThreshold = Math.max(180, avg * 2.2, avg + 80);
  const anomaly =
    lossRate >= 1
      ? `丢 ${lossRate.toFixed(lossRate >= 10 ? 0 : 1)}%`
      : max >= peakThreshold
        ? `峰 ${max}ms`
        : jitter >= 60
          ? `抖 ${Math.round(jitter)}ms`
          : null;

  return { task, latest, avg, max, lossRate, jitter, anomaly };
};

const latencyColor = (latency: number | null, anomaly: string | null) => {
  if (latency === null) return "red";
  if (anomaly) return "orange";
  if (latency > 200) return "orange";
  if (latency > 100) return "yellow";
  return "green";
};

/**
 * 延迟 badge 行：固定单行 + 横向滚动，不会因目标多而破版。
 * 默认显示节点的所有探测目标。
 */
const LatencyBadges = ({ uuid }: { uuid: string }) => {
  const { call } = useRPC2Call();
  const [data, setData] = useState<PingResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    const load = async () => {
      try {
        const result = await call<any, PingResponse>("common:getRecords", {
          uuid,
          type: "ping",
          hours: 24,
        });
        if (!active) return;
        setData(result || {});
        setError(null);
      } catch (err: any) {
        if (!active) return;
        setError(err?.message || "延迟数据读取失败");
      }
    };

    load();
    const timer = window.setInterval(load, 30000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [call, uuid]);

  const summaries = useMemo(() => {
    const records = data?.records || [];
    const tasks = data?.tasks || [];
    const recordsByTask = new Map<number, PingRecord[]>();
    for (const record of records) {
      if (!recordsByTask.has(record.task_id)) {
        recordsByTask.set(record.task_id, []);
      }
      recordsByTask.get(record.task_id)!.push(record);
    }
    return tasks
      .map((task) => analyzeTask(task, recordsByTask.get(task.id) || []))
      .filter((item): item is Summary => Boolean(item))
      .sort((a, b) => {
        // 异常优先 → 然后按延迟升序（小的更好）
        if (a.anomaly && !b.anomaly) return -1;
        if (!a.anomaly && b.anomaly) return 1;
        return (a.latest ?? Infinity) - (b.latest ?? Infinity);
      });
  }, [data]);

  if (error) {
    return (
      <Flex align="center" gap="1" className="vpsnw-latency">
        <Text size="1" color="gray" className="flex items-center gap-1 shrink-0">
          <Activity size={12} />
          延迟
        </Text>
        <Badge color="gray" variant="soft" size="1">
          <span className="text-xs">读取失败</span>
        </Badge>
      </Flex>
    );
  }

  if (!summaries.length) {
    return (
      <Flex align="center" gap="1" className="vpsnw-latency">
        <Text size="1" color="gray" className="flex items-center gap-1 shrink-0">
          <Activity size={12} />
          延迟
        </Text>
        <Badge color="gray" variant="soft" size="1">
          <span className="text-xs">暂无数据</span>
        </Badge>
      </Flex>
    );
  }

  return (
    <Flex
      align="center"
      gap="2"
      className="vpsnw-latency"
      style={{ minWidth: 0 }}
    >
      <Text
        size="1"
        color="gray"
        className="flex items-center gap-1 shrink-0"
      >
        <Activity size={12} />
        延迟
      </Text>
      <div
        className="vpsnw-latency-scroll"
        style={{
          display: "flex",
          gap: 4,
          overflowX: "auto",
          overflowY: "hidden",
          flexWrap: "nowrap",
          paddingBottom: 2,
          scrollbarWidth: "none",
        }}
      >
        {summaries.map((summary) => (
          <Badge
            key={summary.task.id}
            color={latencyColor(summary.latest, summary.anomaly) as any}
            variant="soft"
            size="1"
            title={`${summary.task.name} avg ${summary.avg}ms · max ${summary.max}ms · loss ${summary.lossRate.toFixed(1)}%`}
            style={{ flexShrink: 0, whiteSpace: "nowrap" }}
          >
            <span className="text-xs flex items-center gap-1">
              {summary.anomaly && <AlertTriangle size={10} />}
              <span>
                {summary.task.name}{" "}
                {summary.latest === null ? "·丢包" : `${summary.latest}ms`}
              </span>
            </span>
          </Badge>
        ))}
      </div>
    </Flex>
  );
};

export default LatencyBadges;
