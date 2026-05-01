import { useEffect, useMemo, useRef, useState } from "react";
import * as echarts from "echarts/core";
import { LineChart, type LineSeriesOption } from "echarts/charts";
import { CanvasRenderer } from "echarts/renderers";
import {
  GridComponent,
  TooltipComponent,
  type GridComponentOption,
  type TooltipComponentOption,
} from "echarts/components";
import ReactECharts from "echarts-for-react/lib/core";
import { useRPC2Call } from "@/contexts/RPC2Context";

echarts.use([LineChart, CanvasRenderer, GridComponent, TooltipComponent]);

type Option = echarts.ComposeOption<
  LineSeriesOption | GridComponentOption | TooltipComponentOption
>;

type FlatRecord = {
  time: string;
  cpu: number;
  ram: number;
  ram_total: number;
  disk: number;
  net_in: number;
  net_out: number;
};

type RecentResp = {
  count: number;
  records: FlatRecord[];
};

// 单进程缓存：30 秒节流 + uuid 去重，避免每张卡片重复拉数据
type CacheEntry = { fetchedAt: number; promise: Promise<FlatRecord[]> };
const recentCache = new Map<string, CacheEntry>();
const CACHE_TTL_MS = 30_000;

const fetchRecent = (
  uuid: string,
  call: <Req, Resp>(method: string, params?: Req) => Promise<Resp>,
): Promise<FlatRecord[]> => {
  const now = Date.now();
  const cached = recentCache.get(uuid);
  if (cached && now - cached.fetchedAt < CACHE_TTL_MS) {
    return cached.promise;
  }
  const promise = call<{ uuid: string }, RecentResp>(
    "common:getNodeRecentStatus",
    { uuid },
  )
    .then((resp) => resp?.records ?? [])
    .catch(() => [] as FlatRecord[]);
  recentCache.set(uuid, { fetchedAt: now, promise });
  return promise;
};

interface SparklineProps {
  uuid: string;
  field: "cpu" | "ram" | "load";
  /** Used to compute percentage when field === "ram" (ram_total varies per record). Optional. */
  ramTotal?: number;
  width?: number | string;
  height?: number | string;
  color?: string;
}

const seriesValueFromRecord = (
  rec: FlatRecord,
  field: SparklineProps["field"],
): number => {
  switch (field) {
    case "cpu":
      return rec.cpu ?? 0;
    case "ram":
      return rec.ram_total > 0 ? (rec.ram / rec.ram_total) * 100 : 0;
    case "load":
      // Load1 stored on the cpu series isn't ideal, but recent endpoint exposes it via "load" field; fall back to 0
      return (rec as unknown as { load?: number }).load ?? 0;
    default:
      return 0;
  }
};

const Sparkline = ({
  uuid,
  field,
  width = 80,
  height = 24,
  color,
}: SparklineProps) => {
  const { call } = useRPC2Call();
  const [series, setSeries] = useState<number[]>([]);
  const chartRef = useRef<ReactECharts | null>(null);

  useEffect(() => {
    let active = true;
    const load = async () => {
      const records = await fetchRecent(uuid, call);
      if (!active) return;
      const values = records.map((rec) => seriesValueFromRecord(rec, field));
      setSeries(values);
    };
    load();
    const timer = window.setInterval(load, CACHE_TTL_MS);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [uuid, field, call]);

  const option = useMemo<Option>(() => {
    const stroke = color ?? (field === "cpu" ? "#3b82f6" : field === "ram" ? "#10b981" : "#a855f7");
    return {
      grid: { top: 2, right: 2, bottom: 2, left: 2, containLabel: false },
      xAxis: { type: "category", show: false, boundaryGap: false },
      yAxis: { type: "value", show: false, scale: true },
      tooltip: { show: false },
      animation: false,
      series: [
        {
          type: "line",
          data: series,
          smooth: true,
          symbol: "none",
          lineStyle: { width: 1.4, color: stroke },
          areaStyle: { opacity: 0.18, color: stroke },
        },
      ],
    };
  }, [series, field, color]);

  if (!series.length) {
    return (
      <div
        style={{
          width,
          height,
          display: "inline-block",
          opacity: 0.25,
        }}
      />
    );
  }

  return (
    <ReactECharts
      ref={(instance) => {
        chartRef.current = instance;
      }}
      echarts={echarts}
      option={option}
      style={{ width, height }}
      opts={{ renderer: "canvas" }}
      notMerge
      lazyUpdate
    />
  );
};

export default Sparkline;
