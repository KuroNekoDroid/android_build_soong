// Copyright 2016 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"runtime"
	"strings"

	"android/soong/android"
	"android/soong/remoteexec"
)

var (
	pctx         = android.NewPackageContext("android/soong/cc/config")
	exportedVars = android.NewExportedVariables(pctx)

	// Flags used by lots of devices.  Putting them in package static variables
	// will save bytes in build.ninja so they aren't repeated for every file
	commonGlobalCflags = []string{
		"-DANDROID",
		"-fmessage-length=0",
		"-W",
		"-Wall",
		"-Wno-unused",
		"-Winit-self",
		"-Wpointer-arith",
		"-Wunreachable-code-loop-increment",

		// Make paths in deps files relative
		"-no-canonical-prefixes",

		"-DNDEBUG",
		"-UDEBUG",

		"-fno-exceptions",
		"-Wno-multichar",

		"-O3",
		"-fdebug-default-version=5",

		"-fno-strict-aliasing",

		"-Werror=date-time",
		"-Werror=pragma-pack",
		"-Werror=pragma-pack-suspicious-include",
		"-Werror=string-plus-int",
		"-Werror=unreachable-code-loop-increment",

		// Force deprecation warnings to be warnings for code that compiles with -Werror.
		// Making deprecated usages an error causes extreme pain when trying to deprecate anything.
		"-Wno-error=deprecated-declarations",

		"-D__compiler_offsetof=__builtin_offsetof",

		// Emit address-significance table which allows linker to perform safe ICF. Clang does
		// not emit the table by default on Android since NDK still uses GNU binutils.
		"-faddrsig",

		// Turn on -fcommon explicitly, since Clang now defaults to -fno-common. The cleanup bug
		// tracking this is http://b/151457797.
		"-fcommon",

		// Help catch common 32/64-bit errors.
		"-Werror=int-conversion",

		// Disable overly aggressive warning for macros defined with a leading underscore
		// This happens in AndroidConfig.h, which is included nearly everywhere.
		// TODO: can we remove this now?
		"-Wno-reserved-id-macro",

		// Force clang to always output color diagnostics. Ninja will strip the ANSI
		// color codes if it is not running in a terminal.
		"-fcolor-diagnostics",

		// Warnings from clang-7.0
		"-Wno-sign-compare",

		// Disable -Winconsistent-missing-override until we can clean up the existing
		// codebase for it.
		"-Wno-inconsistent-missing-override",

		// Warnings from clang-10
		// Nested and array designated initialization is nice to have.
		"-Wno-c99-designator",

		// Many old files still have GNU designator syntax.
		"-Wno-gnu-designator",

		// Warnings from clang-12
		"-Wno-gnu-folding-constant",

		// Calls to the APIs that are newer than the min sdk version of the caller should be
		// guarded with __builtin_available.
		"-Wunguarded-availability",
		// This macro allows the bionic versioning.h to indirectly determine whether the
		// option -Wunguarded-availability is on or not.
		"-D__ANDROID_UNAVAILABLE_SYMBOLS_ARE_WEAK__",

		"-fdata-sections",
		"-ffunction-sections",
		"-fno-exceptions",
	}

	commonGlobalConlyflags = []string{}

	commonGlobalAsflags = []string{
		"-D__ASSEMBLY__",
		// TODO(b/235105792): override global -fdebug-default-version=5, it is causing $TMPDIR to
		// end up in the dwarf data for crtend_so.S.
		"-fdebug-default-version=4",
	}

	deviceGlobalCflags = []string{
		"-ffunction-sections",
		"-fdata-sections",
		"-fno-short-enums",
		"-funwind-tables",
		"-fstack-protector-strong",
		"-Wa,--noexecstack",
		"-D_FORTIFY_SOURCE=2",

		"-Wstrict-aliasing=2",

		"-Werror=return-type",
		"-Werror=non-virtual-dtor",
		"-Werror=address",
		"-Werror=sequence-point",
		"-Werror=format-security",
		"-nostdlibinc",

		// Enable Hot Cold splitting optimization.
		"-mllvm -hot-cold-split=true",
	}

	commonGlobalLldflags = []string{
		"-fuse-ld=lld",
		"-Wl,--icf=safe",
	}

	deviceGlobalCppflags = []string{
		"-fvisibility-inlines-hidden",
	}

	deviceGlobalLdflags = []string{
		"-Wl,-z,noexecstack",
		"-Wl,-z,relro",
		"-Wl,-z,now",
		"-Wl,--build-id=md5",
		"-Wl,--fatal-warnings",
		"-Wl,--no-undefined-version",
		// TODO: Eventually we should link against a libunwind.a with hidden symbols, and then these
		// --exclude-libs arguments can be removed.
		"-Wl,--exclude-libs,libgcc.a",
		"-Wl,--exclude-libs,libgcc_stripped.a",
		"-Wl,--exclude-libs,libunwind_llvm.a",
		"-Wl,--exclude-libs,libunwind.a",
		"-Wl,--gc-sections",
	}

	deviceGlobalLldflags = append(deviceGlobalLdflags, commonGlobalLldflags...)

	hostGlobalCflags = []string{}

	hostGlobalCppflags = []string{}

	hostGlobalLdflags = []string{}

	hostGlobalLldflags = commonGlobalLldflags

	commonGlobalCppflags = []string{
		"-Wsign-promo",

		// -Wimplicit-fallthrough is not enabled by -Wall.
		"-Wimplicit-fallthrough",

		// Enable clang's thread-safety annotations in libcxx.
		"-D_LIBCPP_ENABLE_THREAD_SAFETY_ANNOTATIONS",

		// libc++'s math.h has an #include_next outside of system_headers.
		"-Wno-gnu-include-next",
	}

	noOverrideGlobalCflags = []string{
		"-Werror=bool-operation",
		"-Werror=format-insufficient-args",
		"-Werror=implicit-int-float-conversion",
		"-Werror=int-in-bool-context",
		"-Werror=int-to-pointer-cast",
		"-Werror=pointer-to-int-cast",
		"-Werror=xor-used-as-pow",
		// http://b/161386391 for -Wno-void-pointer-to-enum-cast
		"-Wno-void-pointer-to-enum-cast",
		// http://b/161386391 for -Wno-void-pointer-to-int-cast
		"-Wno-void-pointer-to-int-cast",
		// http://b/161386391 for -Wno-pointer-to-int-cast
		"-Wno-pointer-to-int-cast",
		"-Werror=fortify-source",

		"-Werror=address-of-temporary",
		"-Werror=null-dereference",
		"-Werror=return-type",

		// http://b/72331526 Disable -Wtautological-* until the instances detected by these
		// new warnings are fixed.
		"-Wno-tautological-constant-compare",
		"-Wno-tautological-type-limit-compare",
		// http://b/145210666
		"-Wno-reorder-init-list",
		// http://b/145211066
		"-Wno-implicit-int-float-conversion",
		// New warnings to be fixed after clang-r377782.
		"-Wno-tautological-overlap-compare", // http://b/148815696
		// New warnings to be fixed after clang-r383902.
		"-Wno-deprecated-copy",                      // http://b/153746672
		"-Wno-range-loop-construct",                 // http://b/153747076
		"-Wno-zero-as-null-pointer-constant",        // http://b/68236239
		"-Wno-deprecated-anon-enum-enum-conversion", // http://b/153746485
		"-Wno-pessimizing-move",                     // http://b/154270751
		// New warnings to be fixed after clang-r399163
		"-Wno-non-c-typedef-for-linkage", // http://b/161304145
		// New warnings to be fixed after clang-r428724
		"-Wno-align-mismatch", // http://b/193679946
		// New warnings to be fixed after clang-r433403
		"-Wno-error=unused-but-set-variable",  // http://b/197240255
		"-Wno-error=unused-but-set-parameter", // http://b/197240255
		// New warnings to be fixed after clang-r468909
		"-Wno-error=deprecated-builtins", // http://b/241601211
		"-Wno-error=deprecated",          // in external/googletest/googletest
		// New warnings to be fixed after clang-r475365
		"-Wno-error=single-bit-bitfield-constant-conversion", // http://b/243965903
		"-Wno-error=enum-constexpr-conversion",               // http://b/243964282
		"-Wno-error=deprecated-non-prototype",
		"-Wno-error=strict-prototypes",
		"-Wno-error=enum-conversion",
		"-Wno-error=unused-value",
		"-Wno-error=single-bit-bitfield-constant-conversion",
		"-Wno-error=unused-private-field",
		"-Wno-unused-command-line-argument",
		"-Wno-error=thread-safety-analysis",
		"-Wno-error=unguarded-availability",
		"-Wno-error=logical-op-parentheses",
		"-Wno-error=shadow-uncaptured-local",
	}

	noOverride64GlobalCflags = []string{}

	noOverrideExternalGlobalCflags = []string{
		// http://b/191699019
		"-Wno-format-insufficient-args",
		"-Wno-sizeof-array-div",
		"-Wno-incompatible-function-pointer-types",
		"-Wno-unused-but-set-variable",
		"-Wno-unused-but-set-parameter",
		"-Wno-unqualified-std-cast-call",
		"-Wno-bitwise-instead-of-logical",
		"-Wno-misleading-indentation",
		"-Wno-array-parameter",
		"-Wno-gnu-offsetof-extensions",
	}

	// Extra cflags for external third-party projects to disable warnings that
	// are infeasible to fix in all the external projects and their upstream repos.
	extraExternalCflags = []string{
		"-Wno-enum-compare",
		"-Wno-enum-compare-switch",

		// http://b/72331524 Allow null pointer arithmetic until the instances detected by
		// this new warning are fixed.
		"-Wno-null-pointer-arithmetic",

		// Bug: http://b/29823425 Disable -Wnull-dereference until the
		// new instances detected by this warning are fixed.
		"-Wno-null-dereference",

		// http://b/145211477
		"-Wno-pointer-compare",
		"-Wno-final-dtor-non-final-class",

		// http://b/165945989
		"-Wno-psabi",

		// http://b/199369603
		"-Wno-null-pointer-subtraction",

		// http://b/175068488
		"-Wno-string-concatenation",

		// http://b/239661264
		"-Wno-deprecated-non-prototype",
	}

	llvmNextExtraCommonGlobalCflags = []string{
		// Do not report warnings when testing with the top of trunk LLVM.
		"-Wno-error",
	}

	IllegalFlags = []string{
		"-w",
	}

	CStdVersion               = "gnu11"
	CppStdVersion             = "gnu++17"
	ExperimentalCStdVersion   = "gnu17"
	ExperimentalCppStdVersion = "gnu++2a"

	// prebuilts/clang default settings.
	ClangDefaultBase         = "prebuilts/clang/host"
	ClangDefaultVersion      = "clang-r498229b"
	ClangDefaultShortVersion = "17"

	// Directories with warnings from Android.bp files.
	WarningAllowedProjects = []string{
		"device/",
		"vendor/",
	}

	VersionScriptFlagPrefix = "-Wl,--version-script,"

	VisibilityHiddenFlag  = "-fvisibility=hidden"
	VisibilityDefaultFlag = "-fvisibility=default"
)

// BazelCcToolchainVars generates bzl file content containing variables for
// Bazel's cc_toolchain configuration.
func BazelCcToolchainVars(config android.Config) string {
	return android.BazelToolchainVars(config, exportedVars)
}

func ExportStringList(name string, value []string) {
	exportedVars.ExportStringList(name, value)
}

func init() {
	if runtime.GOOS == "linux" {
		commonGlobalCflags = append(commonGlobalCflags, "-fdebug-prefix-map=/proc/self/cwd=")
	}

	exportedVars.ExportStringListStaticVariable("CommonGlobalConlyflags", commonGlobalConlyflags)
	exportedVars.ExportStringListStaticVariable("CommonGlobalAsflags", commonGlobalAsflags)
	exportedVars.ExportStringListStaticVariable("DeviceGlobalCppflags", deviceGlobalCppflags)
	exportedVars.ExportStringListStaticVariable("DeviceGlobalLdflags", deviceGlobalLdflags)
	exportedVars.ExportStringListStaticVariable("DeviceGlobalLldflags", deviceGlobalLldflags)
	exportedVars.ExportStringListStaticVariable("HostGlobalCppflags", hostGlobalCppflags)
	exportedVars.ExportStringListStaticVariable("HostGlobalLdflags", hostGlobalLdflags)
	exportedVars.ExportStringListStaticVariable("HostGlobalLldflags", hostGlobalLldflags)

	// Export the static default CommonGlobalCflags to Bazel.
	exportedVars.ExportStringList("CommonGlobalCflags", commonGlobalCflags)

	pctx.VariableFunc("CommonGlobalCflags", func(ctx android.PackageVarContext) string {
		flags := commonGlobalCflags

		// http://b/131390872
		// Automatically initialize any uninitialized stack variables.
		// Prefer zero-init if multiple options are set.
		if ctx.Config().IsEnvTrue("AUTO_ZERO_INITIALIZE") {
			flags = append(flags, "-ftrivial-auto-var-init=zero -enable-trivial-auto-var-init-zero-knowing-it-will-be-removed-from-clang -Wno-unused-command-line-argument")
		} else if ctx.Config().IsEnvTrue("AUTO_PATTERN_INITIALIZE") {
			flags = append(flags, "-ftrivial-auto-var-init=pattern")
		} else if ctx.Config().IsEnvTrue("AUTO_UNINITIALIZE") {
			flags = append(flags, "-ftrivial-auto-var-init=uninitialized")
		} else {
			// Default to zero initialization.
			flags = append(flags, "-ftrivial-auto-var-init=zero -enable-trivial-auto-var-init-zero-knowing-it-will-be-removed-from-clang -Wno-unused-command-line-argument")
		}

		// Workaround for ccache with clang.
		// See http://petereisentraut.blogspot.com/2011/05/ccache-and-clang.html.
		if ctx.Config().IsEnvTrue("USE_CCACHE") {
			flags = append(flags, "-Wno-unused-command-line-argument")
		}

		if ctx.Config().IsEnvTrue("ALLOW_UNKNOWN_WARNING_OPTION") {
			flags = append(flags, "-Wno-error=unknown-warning-option")
		}

		return strings.Join(flags, " ")
	})

	// Export the static default DeviceGlobalCflags to Bazel.
	// TODO(187086342): handle cflags that are set in VariableFuncs.
	exportedVars.ExportStringList("DeviceGlobalCflags", deviceGlobalCflags)

	pctx.VariableFunc("DeviceGlobalCflags", func(ctx android.PackageVarContext) string {
		return strings.Join(deviceGlobalCflags, " ")
	})

	// Export the static default NoOverrideGlobalCflags to Bazel.
	exportedVars.ExportStringList("NoOverrideGlobalCflags", noOverrideGlobalCflags)
	pctx.VariableFunc("NoOverrideGlobalCflags", func(ctx android.PackageVarContext) string {
		flags := noOverrideGlobalCflags
		if ctx.Config().IsEnvTrue("LLVM_NEXT") {
			flags = append(noOverrideGlobalCflags, llvmNextExtraCommonGlobalCflags...)
		}
		return strings.Join(flags, " ")
	})

	exportedVars.ExportStringListStaticVariable("NoOverride64GlobalCflags", noOverride64GlobalCflags)
	exportedVars.ExportStringListStaticVariable("HostGlobalCflags", hostGlobalCflags)
	exportedVars.ExportStringListStaticVariable("NoOverrideExternalGlobalCflags", noOverrideExternalGlobalCflags)
	exportedVars.ExportStringListStaticVariable("CommonGlobalCppflags", commonGlobalCppflags)
	exportedVars.ExportStringListStaticVariable("ExternalCflags", extraExternalCflags)

	exportedVars.ExportString("CStdVersion", CStdVersion)
	exportedVars.ExportString("CppStdVersion", CppStdVersion)
	exportedVars.ExportString("ExperimentalCStdVersion", ExperimentalCStdVersion)
	exportedVars.ExportString("ExperimentalCppStdVersion", ExperimentalCppStdVersion)

	exportedVars.ExportString("VersionScriptFlagPrefix", VersionScriptFlagPrefix)

	exportedVars.ExportString("VisibilityHiddenFlag", VisibilityHiddenFlag)
	exportedVars.ExportString("VisibilityDefaultFlag", VisibilityDefaultFlag)

	// Everything in these lists is a crime against abstraction and dependency tracking.
	// Do not add anything to this list.
	commonGlobalIncludes := []string{
		"system/core/include",
		"system/logging/liblog/include",
		"system/media/audio/include",
		"hardware/libhardware/include",
		"hardware/libhardware_legacy/include",
		"hardware/ril/include",
		"frameworks/native/include",
		"frameworks/native/opengl/include",
		"frameworks/av/include",
	}
	exportedVars.ExportStringList("CommonGlobalIncludes", commonGlobalIncludes)
	pctx.PrefixedExistentPathsForSourcesVariable("CommonGlobalIncludes", "-I", commonGlobalIncludes)

	pctx.StaticVariableWithEnvOverride("ClangBase", "LLVM_PREBUILTS_BASE", ClangDefaultBase)
	exportedVars.ExportStringStaticVariableWithEnvOverride("ClangVersion", "LLVM_PREBUILTS_VERSION", ClangDefaultVersion)
	pctx.StaticVariable("ClangPath", "${ClangBase}/${HostPrebuiltTag}/${ClangVersion}")
	pctx.StaticVariable("ClangBin", "${ClangPath}/bin")

	exportedVars.ExportStringStaticVariableWithEnvOverride("ClangShortVersion", "LLVM_RELEASE_VERSION", ClangDefaultShortVersion)
	pctx.StaticVariable("ClangAsanLibDir", "${ClangBase}/linux-x86/${ClangVersion}/lib/clang/${ClangShortVersion}/lib/linux")

	// These are tied to the version of LLVM directly in external/llvm, so they might trail the host prebuilts
	// being used for the rest of the build process.
	pctx.SourcePathVariable("RSClangBase", "prebuilts/clang/host")
	pctx.SourcePathVariable("RSClangVersion", "clang-3289846")
	pctx.SourcePathVariable("RSReleaseVersion", "3.8")
	pctx.StaticVariable("RSLLVMPrebuiltsPath", "${RSClangBase}/${HostPrebuiltTag}/${RSClangVersion}/bin")
	pctx.StaticVariable("RSIncludePath", "${RSLLVMPrebuiltsPath}/../lib64/clang/${RSReleaseVersion}/include")

	pctx.PrefixedExistentPathsForSourcesVariable("RsGlobalIncludes", "-I",
		[]string{
			"external/clang/lib/Headers",
			"frameworks/rs/script_api/include",
		})

	pctx.VariableFunc("CcWrapper", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("CC_WRAPPER"); override != "" {
			return override + " "
		}
		return ""
	})

	pctx.StaticVariableWithEnvOverride("RECXXPool", "RBE_CXX_POOL", remoteexec.DefaultPool)
	pctx.StaticVariableWithEnvOverride("RECXXLinksPool", "RBE_CXX_LINKS_POOL", remoteexec.DefaultPool)
	pctx.StaticVariableWithEnvOverride("REClangTidyPool", "RBE_CLANG_TIDY_POOL", remoteexec.DefaultPool)
	pctx.StaticVariableWithEnvOverride("RECXXLinksExecStrategy", "RBE_CXX_LINKS_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
	pctx.StaticVariableWithEnvOverride("REClangTidyExecStrategy", "RBE_CLANG_TIDY_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
	pctx.StaticVariableWithEnvOverride("REAbiDumperExecStrategy", "RBE_ABI_DUMPER_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
	pctx.StaticVariableWithEnvOverride("REAbiLinkerExecStrategy", "RBE_ABI_LINKER_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
}

var HostPrebuiltTag = exportedVars.ExportVariableConfigMethod("HostPrebuiltTag", android.Config.PrebuiltOS)

func ClangPath(ctx android.PathContext, file string) android.SourcePath {
	type clangToolKey string

	key := android.NewCustomOnceKey(clangToolKey(file))

	return ctx.Config().OnceSourcePath(key, func() android.SourcePath {
		return clangPath(ctx).Join(ctx, file)
	})
}

var clangPathKey = android.NewOnceKey("clangPath")

func clangPath(ctx android.PathContext) android.SourcePath {
	return ctx.Config().OnceSourcePath(clangPathKey, func() android.SourcePath {
		clangBase := ClangDefaultBase
		if override := ctx.Config().Getenv("LLVM_PREBUILTS_BASE"); override != "" {
			clangBase = override
		}
		clangVersion := ClangDefaultVersion
		if override := ctx.Config().Getenv("LLVM_PREBUILTS_VERSION"); override != "" {
			clangVersion = override
		}
		return android.PathForSource(ctx, clangBase, ctx.Config().PrebuiltOS(), clangVersion)
	})
}
