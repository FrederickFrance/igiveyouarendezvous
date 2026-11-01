# igiveyouarendezvous

[![CI](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml/badge.svg)](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/FrederickFrance/igiveyouarendezvous.svg)](https://pkg.go.dev/github.com/FrederickFrance/igiveyouarendezvous)

[English](README.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Italiano](README.it.md) · **ไทย** · [tlhIngan Hol](README.tlh.md) · [Ch'ti](README.pcd.md)

โปรแกรมจัดตารางสัมภาษณ์ระหว่างผู้สมัครกับผู้สัมภาษณ์ เขียนด้วยภาษา Go
ใช้ได้ทั้งแบบคำสั่งและแบบไลบรารี

ผู้สมัครแต่ละคนจะได้พบผู้สัมภาษณ์ทุกคนของตนในวันเดียวกัน
โดยการสัมภาษณ์ต่อเนื่องกัน ตารางจะทำให้มีผู้สมัครที่ได้พบผู้สัมภาษณ์ครบทุกคนมากที่สุดก่อน
(ตามลำดับความสำคัญ) จากนั้นให้มีการพบกันมากที่สุด แล้วจึงประหยัดเวลาของผู้สัมภาษณ์
การรันหนึ่งครั้งจัดตารางหนึ่งวัน ส่วนที่จัดไม่ได้จะถูกเขียนลงในไฟล์โจทย์ของวันถัดไป

## ตารางแรกของคุณ ทีละขั้นตอน

ไม่ต้องเขียนโปรแกรม ไม่ต้องรู้จักการพิมพ์คำสั่ง: ทำตามขั้นตอนต่อไปนี้ตามลำดับทุกประการ

**1. ดาวน์โหลดโปรแกรม** เปิดหน้า [Releases](https://github.com/FrederickFrance/igiveyouarendezvous/releases/latest)
แล้วดาวน์โหลดไฟล์ zip ที่ตรงกับคอมพิวเตอร์ของคุณ:

| คอมพิวเตอร์ของคุณ | ไฟล์ |
|---|---|
| Windows | `igiveyouarendezvous-windows.zip` |
| Mac ชิป Apple (M1, M2…: เมนู Apple > About This Mac) | `igiveyouarendezvous-mac-apple.zip` |
| Mac ชิป Intel | `igiveyouarendezvous-mac-intel.zip` |
| Linux | `igiveyouarendezvous-linux.zip` |

แตกไฟล์ zip (บน Mac ดับเบิลคลิก บน Windows คลิกขวา > Extract All)
คุณจะได้โฟลเดอร์ที่มีโปรแกรมและไฟล์ชื่อ `problem.yaml`

**2. อธิบายวันของคุณ** เปิด `problem.yaml` ด้วยโปรแกรมแก้ไขข้อความธรรมดา:
บน Windows คลิกขวา > Open with > Notepad บน Mac คลิกขวา > Open With > TextEdit
ใส่ชื่อผู้สัมภาษณ์และผู้สมัครของคุณแทนชื่อในตัวอย่าง ใต้ผู้สมัครแต่ละคน `meet:`
คือรายชื่อผู้สัมภาษณ์ที่ต้องพบ อยู่ในวงเล็บเหลี่ยมและคั่นด้วยจุลภาค
กฎสามข้อ: เว้นวรรคต้นบรรทัดให้เหมือนเดิมทุกประการ (เว้นวรรคสองครั้ง ห้ามใช้ปุ่ม Tab)
เขียนชื่อแต่ละชื่อให้เหมือนกันทุกที่ และอย่าลบเครื่องหมายทวิภาค (:) แล้วบันทึกไฟล์

```yaml
day: 09:00-17:00
breaks: 12:00-13:30
duration: 1h

recruiters:
  Alice:
  Bob:

candidates:
  David:
    meet: [Alice, Bob]
  Emma:
    meet: [Bob]
```

**3. เปิดหน้าต่างคำสั่ง (terminal) ในโฟลเดอร์นั้น**
- Windows: เปิดโฟลเดอร์ คลิกที่แถบที่อยู่ด้านบน พิมพ์ `cmd` แล้วกด Enter
  จะมีหน้าต่างสีดำเปิดขึ้น
- Mac: เปิดแอป Terminal (Applications > Utilities) พิมพ์ `cd ` (มีเว้นวรรค)
  ลากโฟลเดอร์มาวางในหน้าต่าง แล้วกด Enter
- Linux: คลิกขวาในโฟลเดอร์ > Open in Terminal

**4. รันโปรแกรม** พิมพ์บรรทัดนี้ แล้วกด Enter:
- Windows: `igiveyouarendezvous problem.yaml`
- Mac และ Linux: `./igiveyouarendezvous problem.yaml`

ภายในหนึ่งนาที ตารางจะปรากฏ: ใครพบใคร เวลาใด บน Mac ครั้งแรกระบบจะบล็อกโปรแกรมที่ไม่รู้จัก
ให้เปิด System Settings > Privacy & Security คลิก Open Anyway ด้านล่าง
แล้วพิมพ์บรรทัดเดิมอีกครั้ง (Windows อาจแสดงคำเตือนเช่นกัน: More info > Run anyway)

**5. เก็บผลลัพธ์** หากต้องการผลลัพธ์เป็นไฟล์แทนการแสดงในหน้าต่าง ให้พิมพ์
`igiveyouarendezvous problem.yaml > schedule.txt` (Mac และ Linux ขึ้นต้นด้วย `./`)
แล้วเปิด `schedule.txt` เหมือนขั้นตอนที่ 2

**6. วันถัดไป** สำหรับผู้ที่จัดลงตารางวันนี้ไม่ได้ ให้พิมพ์
`igiveyouarendezvous -rest tomorrow.yaml problem.yaml` (Mac และ Linux ขึ้นต้นด้วย `./`)
โปรแกรมจะเขียนไฟล์ `tomorrow.yaml` ด้วย: เป็นไฟล์เดิมที่เหลือเฉพาะการพบที่ยังไม่ได้จัด
เพิ่มเติมได้ถ้าจำเป็น แล้วใช้ไฟล์นี้ในวันถัดไปแทน `problem.yaml`

**หากมีข้อผิดพลาด** โปรแกรมจะบอกว่าบรรทัดใดในไฟล์ผิด
ให้ตรวจการเว้นวรรคต้นบรรทัด เครื่องหมายทวิภาค และชื่อที่ต้องเขียนเหมือนใต้ `recruiters`
ตัวอย่างฉบับเต็มพร้อมคำอธิบาย `example-full.yaml` แสดงตัวเลือกอื่นทั้งหมด:
เวลาที่ว่างรายบุคคลและรายวัน ลำดับความสำคัญ การสัมภาษณ์ร่วม…

## การติดตั้ง

Go 1.27.3 ขึ้นไป ไม่ต้องใช้ C:

```sh
go install github.com/FrederickFrance/igiveyouarendezvous/cmd/igiveyouarendezvous@latest
```

## การใช้งาน

```sh
igiveyouarendezvous -date 2026-11-03 -rest day2.yaml example/problem.yaml
```

ไฟล์โจทย์ (YAML หรือ JSON) ระบุผู้สัมภาษณ์ ผู้สมัครพร้อมผู้สัมภาษณ์ที่ต้องพบ
เวลาที่ว่าง และการตั้งค่า ดู[ตัวอย่างพร้อมคำอธิบาย](example/problem.yaml)
และ [JSON schema](schema/problem.schema.json) กด Ctrl+C
เพื่อหยุดการค้นหาและแสดงตารางที่ดีที่สุดที่พบจนถึงตอนนั้น

เอกสารฉบับเต็มมีเป็น[ภาษาอังกฤษ](README.md)และ[ภาษาฝรั่งเศส](README.fr.md)

## สัญญาอนุญาต

เลือกได้ระหว่าง [0BSD](LICENSE) หรือ [WTFPL](WTFPL.txt) รุ่นที่ 2
