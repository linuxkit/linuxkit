/*
 * bhyve DSDT template
 */
DefinitionBlock ("bhyve_dsdt.aml", "DSDT", 2,"BHYVE ", "BVDSDT  ", 0x00000001)
{
  Name (_S5, Package ()
  {
      0x05,
      Zero,
  })
  Name (PICM, 0x00)
  Method (_PIC, 1, NotSerialized)
  {
    Store (Arg0, PICM)
  }

  Scope (_SB)
  {
    Device (PC00)
    {
      Name (_HID, EisaId ("PNP0A03"))
      Name (_ADR, Zero)
      Method (_BBN, 0, NotSerialized)
      {
          Return (0x00000000)
      }
      Name (_CRS, ResourceTemplate ()
      {
        WordBusNumber (ResourceProducer, MinFixed, MaxFixed, PosDecode,
          0x0000,             // Granularity
          0x0000,             // Range Minimum
          0x0000,             // Range Maximum
          0x0000,             // Translation Offset
          0x0001,             // Length
          ,, )
        IO (Decode16,
          0x0CF8,             // Range Minimum
          0x0CF8,             // Range Maximum
          0x01,               // Alignment
          0x08,               // Length
          )
        WordIO (ResourceProducer, MinFixed, MaxFixed, PosDecode, EntireRange,
          0x0000,             // Granularity
          0x0000,             // Range Minimum
          0x0CF7,             // Range Maximum
          0x0000,             // Translation Offset
          0x0CF8,             // Length
          ,, , TypeStatic)
        WordIO (ResourceProducer, MinFixed, MaxFixed, PosDecode, EntireRange,
          0x0000,             // Granularity
          0x0D00,             // Range Minimum
          0x1FFF,             // Range Maximum
          0x0000,             // Translation Offset
          0x1300,             // Length
          ,, , TypeStatic)
        WordIO (ResourceProducer, MinFixed, MaxFixed, PosDecode, EntireRange,
          0x0000,             // Granularity
          0x0000,             // Range Minimum
          0x0000,             // Range Maximum
          0x0000,             // Translation Offset
          0x0020,             // Length
          ,, , TypeStatic)
        DWordMemory (ResourceProducer, PosDecode, MinFixed, MaxFixed, NonCacheable, ReadWrite,
          0x00000000,         // Granularity
          0x00000000,         // Range Minimum

          0x00000000,         // Range Maximum

          0x00000000,         // Translation Offset
          0x00000000,         // Length

          ,, , AddressRangeMemory, TypeStatic)
        QWordMemory (ResourceProducer, PosDecode, MinFixed, MaxFixed, NonCacheable, ReadWrite,
          0x0000000000000000, // Granularity
          0x0000000000000000, // Range Minimum

          0x0000000000000000, // Range Maximum

          0x0000000000000000, // Translation Offset
          0x0000000000000000, // Length

          ,, , AddressRangeMemory, TypeStatic)
      })
      Name (PPRT, Package ()
      {
        Package ()
        {
          0x1FFFF,
          0x00,
          \_SB.PC00.ISA.LNKA,,
          0x00
        },
        Package ()
        {
          0x2FFFF,
          0x00,
          \_SB.PC00.ISA.LNKB,,
          0x00
        },
        Package ()
        {
          0x3FFFF,
          0x00,
          \_SB.PC00.ISA.LNKC,,
          0x00
        },
        Package ()
        {
          0x4FFFF,
          0x00,
          \_SB.PC00.ISA.LNKD,,
          0x00
        },
        Package ()
        {
          0x5FFFF,
          0x00,
          \_SB.PC00.ISA.LNKE,,
          0x00
        },
        Package ()
        {
          0x6FFFF,
          0x00,
          \_SB.PC00.ISA.LNKF,,
          0x00
        },
        Package ()
        {
          0x7FFFF,
          0x00,
          \_SB.PC00.ISA.LNKG,,
          0x00
        },
        Package ()
        {
          0x8FFFF,
          0x00,
          \_SB.PC00.ISA.LNKH,,
          0x00
        },
      })
      Name (APRT, Package ()
      {
        Package ()
        {
          0x1FFFF,
          0x00,
          Zero,
          0x10
        },
        Package ()
        {
          0x2FFFF,
          0x00,
          Zero,
          0x11
        },
        Package ()
        {
          0x3FFFF,
          0x00,
          Zero,
          0x12
        },
        Package ()
        {
          0x4FFFF,
          0x00,
          Zero,
          0x13
        },
        Package ()
        {
          0x5FFFF,
          0x00,
          Zero,
          0x14
        },
        Package ()
        {
          0x6FFFF,
          0x00,
          Zero,
          0x15
        },
        Package ()
        {
          0x7FFFF,
          0x00,
          Zero,
          0x16
        },
        Package ()
        {
          0x8FFFF,
          0x00,
          Zero,
          0x17
        },
      })
      Method (_PRT, 0, NotSerialized)
      {
        If (PICM)
        {
          Return (APRT)
        }
        Else
        {
          Return (PPRT)
        }
      }

      Device (ISA)
      {
        Name (_ADR, 0x001F0000)
        OperationRegion (LPCR, PCI_Config, 0x00, 0x100)
        Field (LPCR, AnyAcc, NoLock, Preserve)
        {
          Offset (0x60),
          PIRA,   8,
          PIRB,   8,
          PIRC,   8,
          PIRD,   8,
          Offset (0x68),
          PIRE,   8,
          PIRF,   8,
          PIRG,   8,
          PIRH,   8
        }


        Method (PIRV, 1, NotSerialized)
        {
          If (And (Arg0, 0x80))
          {
            Return (0x00)
          }
          And (Arg0, 0x0F, Local0)
          If (LLess (Local0, 0x03))
          {
            Return (0x00)
          }
          If (LEqual (Local0, 0x08))
          {
            Return (0x00)
          }
          If (LEqual (Local0, 0x0D))
          {
            Return (0x00)
          }
          Return (0x01)
        }

        Device (LNKA)
        {
          Name (_HID, EisaId ("PNP0C0F"))
          Name (_UID, 0x01)
          Method (_STA, 0, NotSerialized)
          {
            If (PIRV (PIRA))
            {
               Return (0x0B)
            }
            Else
            {
               Return (0x09)
            }
          }
          Name (_PRS, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {3,4,5,6,7,9,10,11,12,14,15}
          })
          Name (CB01, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {}
          })
          CreateWordField (CB01, 0x01, CIRA)
          Method (_CRS, 0, NotSerialized)
          {
            And (PIRA, 0x8F, Local0)
            If (PIRV (Local0))
            {
              ShiftLeft (0x01, Local0, CIRA)
            }
            Else
            {
              Store (0x00, CIRA)
            }
            Return (CB01)
          }
          Method (_DIS, 0, NotSerialized)
          {
            Store (0x80, PIRA)
          }
          Method (_SRS, 1, NotSerialized)
          {
            CreateWordField (Arg0, 0x01, SIRA)
            FindSetRightBit (SIRA, Local0)
            Store (Decrement (Local0), PIRA)
          }
        }

        Device (LNKB)
        {
          Name (_HID, EisaId ("PNP0C0F"))
          Name (_UID, 0x02)
          Method (_STA, 0, NotSerialized)
          {
            If (PIRV (PIRB))
            {
               Return (0x0B)
            }
            Else
            {
               Return (0x09)
            }
          }
          Name (_PRS, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {3,4,5,6,7,9,10,11,12,14,15}
          })
          Name (CB02, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {}
          })
          CreateWordField (CB02, 0x01, CIRB)
          Method (_CRS, 0, NotSerialized)
          {
            And (PIRB, 0x8F, Local0)
            If (PIRV (Local0))
            {
              ShiftLeft (0x01, Local0, CIRB)
            }
            Else
            {
              Store (0x00, CIRB)
            }
            Return (CB02)
          }
          Method (_DIS, 0, NotSerialized)
          {
            Store (0x80, PIRB)
          }
          Method (_SRS, 1, NotSerialized)
          {
            CreateWordField (Arg0, 0x01, SIRB)
            FindSetRightBit (SIRB, Local0)
            Store (Decrement (Local0), PIRB)
          }
        }

        Device (LNKC)
        {
          Name (_HID, EisaId ("PNP0C0F"))
          Name (_UID, 0x03)
          Method (_STA, 0, NotSerialized)
          {
            If (PIRV (PIRC))
            {
               Return (0x0B)
            }
            Else
            {
               Return (0x09)
            }
          }
          Name (_PRS, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {3,4,5,6,7,9,10,11,12,14,15}
          })
          Name (CB03, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {}
          })
          CreateWordField (CB03, 0x01, CIRC)
          Method (_CRS, 0, NotSerialized)
          {
            And (PIRC, 0x8F, Local0)
            If (PIRV (Local0))
            {
              ShiftLeft (0x01, Local0, CIRC)
            }
            Else
            {
              Store (0x00, CIRC)
            }
            Return (CB03)
          }
          Method (_DIS, 0, NotSerialized)
          {
            Store (0x80, PIRC)
          }
          Method (_SRS, 1, NotSerialized)
          {
            CreateWordField (Arg0, 0x01, SIRC)
            FindSetRightBit (SIRC, Local0)
            Store (Decrement (Local0), PIRC)
          }
        }

        Device (LNKD)
        {
          Name (_HID, EisaId ("PNP0C0F"))
          Name (_UID, 0x04)
          Method (_STA, 0, NotSerialized)
          {
            If (PIRV (PIRD))
            {
               Return (0x0B)
            }
            Else
            {
               Return (0x09)
            }
          }
          Name (_PRS, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {3,4,5,6,7,9,10,11,12,14,15}
          })
          Name (CB04, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {}
          })
          CreateWordField (CB04, 0x01, CIRD)
          Method (_CRS, 0, NotSerialized)
          {
            And (PIRD, 0x8F, Local0)
            If (PIRV (Local0))
            {
              ShiftLeft (0x01, Local0, CIRD)
            }
            Else
            {
              Store (0x00, CIRD)
            }
            Return (CB04)
          }
          Method (_DIS, 0, NotSerialized)
          {
            Store (0x80, PIRD)
          }
          Method (_SRS, 1, NotSerialized)
          {
            CreateWordField (Arg0, 0x01, SIRD)
            FindSetRightBit (SIRD, Local0)
            Store (Decrement (Local0), PIRD)
          }
        }

        Device (LNKE)
        {
          Name (_HID, EisaId ("PNP0C0F"))
          Name (_UID, 0x05)
          Method (_STA, 0, NotSerialized)
          {
            If (PIRV (PIRE))
            {
               Return (0x0B)
            }
            Else
            {
               Return (0x09)
            }
          }
          Name (_PRS, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {3,4,5,6,7,9,10,11,12,14,15}
          })
          Name (CB05, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {}
          })
          CreateWordField (CB05, 0x01, CIRE)
          Method (_CRS, 0, NotSerialized)
          {
            And (PIRE, 0x8F, Local0)
            If (PIRV (Local0))
            {
              ShiftLeft (0x01, Local0, CIRE)
            }
            Else
            {
              Store (0x00, CIRE)
            }
            Return (CB05)
          }
          Method (_DIS, 0, NotSerialized)
          {
            Store (0x80, PIRE)
          }
          Method (_SRS, 1, NotSerialized)
          {
            CreateWordField (Arg0, 0x01, SIRE)
            FindSetRightBit (SIRE, Local0)
            Store (Decrement (Local0), PIRE)
          }
        }

        Device (LNKF)
        {
          Name (_HID, EisaId ("PNP0C0F"))
          Name (_UID, 0x06)
          Method (_STA, 0, NotSerialized)
          {
            If (PIRV (PIRF))
            {
               Return (0x0B)
            }
            Else
            {
               Return (0x09)
            }
          }
          Name (_PRS, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {3,4,5,6,7,9,10,11,12,14,15}
          })
          Name (CB06, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {}
          })
          CreateWordField (CB06, 0x01, CIRF)
          Method (_CRS, 0, NotSerialized)
          {
            And (PIRF, 0x8F, Local0)
            If (PIRV (Local0))
            {
              ShiftLeft (0x01, Local0, CIRF)
            }
            Else
            {
              Store (0x00, CIRF)
            }
            Return (CB06)
          }
          Method (_DIS, 0, NotSerialized)
          {
            Store (0x80, PIRF)
          }
          Method (_SRS, 1, NotSerialized)
          {
            CreateWordField (Arg0, 0x01, SIRF)
            FindSetRightBit (SIRF, Local0)
            Store (Decrement (Local0), PIRF)
          }
        }

        Device (LNKG)
        {
          Name (_HID, EisaId ("PNP0C0F"))
          Name (_UID, 0x07)
          Method (_STA, 0, NotSerialized)
          {
            If (PIRV (PIRG))
            {
               Return (0x0B)
            }
            Else
            {
               Return (0x09)
            }
          }
          Name (_PRS, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {3,4,5,6,7,9,10,11,12,14,15}
          })
          Name (CB07, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {}
          })
          CreateWordField (CB07, 0x01, CIRG)
          Method (_CRS, 0, NotSerialized)
          {
            And (PIRG, 0x8F, Local0)
            If (PIRV (Local0))
            {
              ShiftLeft (0x01, Local0, CIRG)
            }
            Else
            {
              Store (0x00, CIRG)
            }
            Return (CB07)
          }
          Method (_DIS, 0, NotSerialized)
          {
            Store (0x80, PIRG)
          }
          Method (_SRS, 1, NotSerialized)
          {
            CreateWordField (Arg0, 0x01, SIRG)
            FindSetRightBit (SIRG, Local0)
            Store (Decrement (Local0), PIRG)
          }
        }

        Device (LNKH)
        {
          Name (_HID, EisaId ("PNP0C0F"))
          Name (_UID, 0x08)
          Method (_STA, 0, NotSerialized)
          {
            If (PIRV (PIRH))
            {
               Return (0x0B)
            }
            Else
            {
               Return (0x09)
            }
          }
          Name (_PRS, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {3,4,5,6,7,9,10,11,12,14,15}
          })
          Name (CB08, ResourceTemplate ()
          {
            IRQ (Level, ActiveLow, Shared, )
              {}
          })
          CreateWordField (CB08, 0x01, CIRH)
          Method (_CRS, 0, NotSerialized)
          {
            And (PIRH, 0x8F, Local0)
            If (PIRV (Local0))
            {
              ShiftLeft (0x01, Local0, CIRH)
            }
            Else
            {
              Store (0x00, CIRH)
            }
            Return (CB08)
          }
          Method (_DIS, 0, NotSerialized)
          {
            Store (0x80, PIRH)
          }
          Method (_SRS, 1, NotSerialized)
          {
            CreateWordField (Arg0, 0x01, SIRH)
            FindSetRightBit (SIRH, Local0)
            Store (Decrement (Local0), PIRH)
          }
        }

        Device (SIO)
        {
          Name (_HID, EisaId ("PNP0C02"))
          Name (_CRS, ResourceTemplate ()
          {
            IO (Decode16,
              0x0060,             // Range Minimum
              0x0060,             // Range Maximum
              0x01,               // Alignment
              0x01,               // Length
              )
            IO (Decode16,
              0x0064,             // Range Minimum
              0x0064,             // Range Maximum
              0x01,               // Alignment
              0x01,               // Length
              )
            IO (Decode16,
              0x0220,             // Range Minimum
              0x0220,             // Range Maximum
              0x01,               // Alignment
              0x04,               // Length
              )
            IO (Decode16,
              0x0224,             // Range Minimum
              0x0224,             // Range Maximum
              0x01,               // Alignment
              0x04,               // Length
              )
            Memory32Fixed (ReadWrite,
              0xE0000000,         // Address Base
              0x10000000,         // Address Length
              )
            IO (Decode16,
              0x04D0,             // Range Minimum
              0x04D0,             // Range Maximum
              0x01,               // Alignment
              0x02,               // Length
              )
            IO (Decode16,
              0x0061,             // Range Minimum
              0x0061,             // Range Maximum
              0x01,               // Alignment
              0x01,               // Length
              )
            IO (Decode16,
              0x0400,             // Range Minimum
              0x0400,             // Range Maximum
              0x01,               // Alignment
              0x08,               // Length
              )
            IO (Decode16,
              0x00B2,             // Range Minimum
              0x00B2,             // Range Maximum
              0x01,               // Alignment
              0x01,               // Length
              )
            IO (Decode16,
              0x0084,             // Range Minimum
              0x0084,             // Range Maximum
              0x01,               // Alignment
              0x01,               // Length
              )
            IO (Decode16,
              0x0072,             // Range Minimum
              0x0072,             // Range Maximum
              0x01,               // Alignment
              0x06,               // Length
              )
          })
        }

        Device (COM1)
        {
          Name (_HID, EisaId ("PNP0501"))
          Name (_UID, 1)
          Name (_CRS, ResourceTemplate ()
          {
            IO (Decode16,
              0x03F8,             // Range Minimum
              0x03F8,             // Range Maximum
              0x01,               // Alignment
              0x08,               // Length
              )
            IRQNoFlags ()
              {4}
          })
        }

        Device (COM2)
        {
          Name (_HID, EisaId ("PNP0501"))
          Name (_UID, 2)
          Name (_CRS, ResourceTemplate ()
          {
            IO (Decode16,
              0x02F8,             // Range Minimum
              0x02F8,             // Range Maximum
              0x01,               // Alignment
              0x08,               // Length
              )
            IRQNoFlags ()
              {3}
          })
        }

        Device (RTC)
        {
          Name (_HID, EisaId ("PNP0B00"))
          Name (_CRS, ResourceTemplate ()
          {
            IO (Decode16,
              0x0070,             // Range Minimum
              0x0070,             // Range Maximum
              0x01,               // Alignment
              0x02,               // Length
              )
            IRQNoFlags ()
              {8}
          })
        }

        Device (PIC)
        {
          Name (_HID, EisaId ("PNP0000"))
          Name (_CRS, ResourceTemplate ()
          {
            IO (Decode16,
              0x0020,             // Range Minimum
              0x0020,             // Range Maximum
              0x01,               // Alignment
              0x02,               // Length
              )
            IO (Decode16,
              0x00A0,             // Range Minimum
              0x00A0,             // Range Maximum
              0x01,               // Alignment
              0x02,               // Length
              )
            IRQNoFlags ()
              {2}
          })
        }

        Device (TIMR)
        {
          Name (_HID, EisaId ("PNP0100"))
          Name (_CRS, ResourceTemplate ()
          {
            IO (Decode16,
              0x0040,             // Range Minimum
              0x0040,             // Range Maximum
              0x01,               // Alignment
              0x04,               // Length
              )
            IRQNoFlags ()
              {0}
          })
        }
      }
    }
  }

  Scope (_SB.PC00)
  {
    Device (HPET)
    {
      Name (_HID, EISAID("PNP0103"))
      Name (_UID, 0)
      Name (_CRS, ResourceTemplate ()
      {
        Memory32Fixed (ReadWrite,
          0xFED00000,         // Address Base
          0x00000400,         // Address Length
          )
      })
    }
  }
}
