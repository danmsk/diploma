from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ProfileRequest(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, profile: _Optional[_Iterable[int]] = ...) -> None: ...

class ProfileResponse(_message.Message):
    __slots__ = ("recommendations",)
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    recommendations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, recommendations: _Optional[_Iterable[str]] = ...) -> None: ...
