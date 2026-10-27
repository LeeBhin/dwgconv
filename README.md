# dwgconv

DWG ↔ DXF 변환기 (Go, 외부 프로그램·라이브러리 없음)

Open Design Alliance 의 공개 문서 *Open Design Specification for .dwg files* 를 기준으로 구현했습니다.
사양서와 실제 파일이 다른 곳은 실제 파일에 맞췄고, 각 위치에 주석으로 남겨 두었습니다.

## 지원 범위

| DWG 버전 | AutoCAD 버전 | 읽기 | 쓰기 |
|---|---|---|---|
| R2004 (AC1018) | 2004–2006 | ✅ | ✅ |
| R2007 (AC1021) | 2007–2009 | ✅ | ✅ |
| R2010 (AC1024) | 2010–2012 | ✅ | ✅ |
| R2013 (AC1027) | 2013–2017 | ✅ | ✅ |
| R2018 (AC1032) | 2018– | ✅ | ✅ (기본값) |

- DXF 는 ASCII, AC1032 형식으로 씁니다.
- 사양서에 구조가 없는 클래스(AEC_* 등)는 원래 비트를 그대로 보존합니다. DXF 에는 `ACAD_PROXY_OBJECT` 로 담깁니다.

## 사용법

```sh
dwgconv in.dwg out.dxf
dwgconv [-version R2018] in.dxf out.dwg   # R2004 | R2007 | R2010 | R2013 | R2018
```

## 검증

- 디코드 → 인코드가 원본과 비트 단위로 같습니다. 샘플 도면의 모든 객체로 확인했습니다 (`cmd/dwgcheck`).
- DXF 출력과 가져오기는 ODA File Converter 의 결과와 비교했습니다 (`cmd/dxfcheck`).
- 쓴 DWG 는 다음 세 프로그램에서 복구 없이 열리는 것을 확인했습니다.
  - Autodesk DWG TrueView 2027 (`scripts/tvopen.ps1`)
  - ODA File Converter
  - LibreDWG

검증에 쓴 도면 파일은 저장소에 포함하지 않습니다. `.work/orig/sN.dwg` 와 `.work/oda/sN.dxf` 에 직접 넣어서 씁니다.

## 구성

| 경로 | 내용 |
|---|---|
| `internal/container` | 파일 구조. R2004 계열(LZ77)과 R2007(Reed-Solomon, R21 압축) |
| `internal/bits` | 비트 코드 (§2) |
| `internal/dwg` | 헤더, 클래스, 객체. 타입마다 함수 하나가 읽기와 쓰기를 함께 맡습니다 |
| `internal/dxf` | DXF 그룹 읽기·쓰기, 양방향 매퍼 |
| `internal/conv` | DWG ↔ DXF 매핑과 문서 조립 |

DWG 와 DXF 는 Autodesk, Inc. 의 상표입니다. 이 프로젝트는 Autodesk 및 Open Design Alliance 와 관계가 없습니다.
