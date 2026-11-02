// Package bannerlord reads Mount & Blade II: Bannerlord module metadata, computes a valid module
// load order, and reads and writes the launcher's LauncherData.xml.
//
// The formats follow the BUTR tooling, which is the reference for how launchers treat these files:
//
//   - BUTR/Bannerlord.ModuleManager, src/Bannerlord.ModuleManager.Models/ModuleInfoExtended.cs
//     (FromXml: the element and attribute names, the Official/ModuleType rule),
//     LoadType.cs, ApplicationVersion.cs and ApplicationVersionComparer.cs (versions such as
//     "v1.2.3.0", "e1.0.0" or "v1.2.*"), src/Bannerlord.ModuleManager/ModuleInfoExtendedExtensions.cs
//     (how DependedModules, ModulesToLoadAfterThis and the metadata combine) and ModuleSorter.cs.
//   - BUTR/Bannerlord.Module.Template, src/Bannerlord.Module.CSharp/_Module/SubModule.xml, and
//     BUTR/Bannerlord.Harmony, src/Bannerlord.Harmony/_Module/SubModule.xml: real examples.
//   - BUTR/Bannerlord.LauncherManager, src/Bannerlord.LauncherManager/
//     LauncherManagerHandler.Modules.Orderer.cs: the default order of modules the user has not
//     placed (official first, then by id).
//   - BUTR/Bannerlord.BLSE, src/Bannerlord.LauncherEx/BUTRLauncherManagerHandler.cs and
//     Patches/UserDataManagerPatch.cs: LauncherData.xml is the game launcher's XmlSerializer
//     "UserData" with GameType, SingleplayerData and MultiplayerData, each holding ModDatas of
//     UserModData (Id, IsSelected). BLSE appends its own settings elements to the same root. The
//     ModDatas/UserModData element nesting and LastKnownVersion are TaleWorlds' serializer output,
//     which no BUTR file spells out; they are checked only against third-party readers.
//
// All XML input is untrusted: it is size-limited, DOCTYPE declarations are refused, and module
// ids are validated before they are returned. See SECURITY.md.
package bannerlord
